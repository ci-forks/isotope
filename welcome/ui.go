package main

import (
	"image"
	"time"

	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/text"
	"gioui.org/widget"

	"github.com/kairos-community/isotope/guikit"
)

// Hooks are every side effect the UI triggers. main wires them to the
// session; the tests pass fakes.
type Hooks struct {
	// Invalidate asks for a frame from another goroutine.
	Invalidate func()
	// Logout ends the desktop session.
	Logout func() error
	// OpenShortcutSheet opens the full sheet, as Super+F1 does.
	OpenShortcutSheet func() error
	// Finish records how the flow ended and closes the window. The UI
	// calls it once.
	Finish func(Stamp)
}

// page is one screen of the flow. Each lives in its own page_*.go file and
// registers its constructor in pageMakers from an init function; a page
// with no constructor yet is a stubPage. Embed basePage for the usual
// footer and Continue.
type page interface {
	// Body draws the page above the footer, in the whole column. Use
	// u.scrolled so a large scale scrolls instead of cutting it.
	Body(gtx layout.Context) layout.Dimensions
	// Footer is the footer's buttons, left and right, made from the
	// buttons the UI owns.
	Footer(n nav) (leading, trailing []guikit.Action)
	// Update handles the page's own input before it is drawn.
	Update(gtx layout.Context)
	// Continue is the Continue button, and Enter while it has the focus.
	// It moves the flow on; the UI finishes when the flow is done.
	Continue()
}

// nav are the footer buttons the UI owns, already labelled. Back is nil on
// the first page.
type nav struct {
	Back, Skip, Continue *guikit.Action
}

// pageMakers builds the page for an ID; each page file adds itself.
var pageMakers = map[PageID]func(u *UI) page{}

// UI is the Welcome window: the flow, the page on screen and its frame.
type UI struct {
	*guikit.Theme
	flow    *Flow
	desktop Desktop
	hooks   Hooks
	now     func() time.Time

	// version is the sidebar's note, "Isotope v0.0.25 · sway".
	version string
	// shortcuts is the variant's sheet (ShortcutsFile).
	shortcuts []Shortcut
	// appsPending is set when Apps was skipped for being offline: the
	// stamp then asks the next login to remind the user.
	appsPending bool

	pages    map[PageID]page
	shown    PageID    // the page drawn last frame
	pageAt   time.Time // when it came in, for the slide
	focus    event.Tag // focus this on the next frame
	scale    float32   // Ctrl+Plus and Ctrl+Minus
	finished bool

	back, skip, cont widget.Clickable
}

func newUI(flow *Flow, d Desktop, sh *text.Shaper, h Hooks) *UI {
	u := &UI{
		Theme:   &guikit.Theme{Shaper: sh},
		flow:    flow,
		desktop: d,
		hooks:   h,
		now:     time.Now,
		version: "Isotope",
		pages:   map[PageID]page{},
		scale:   1,
	}
	u.focus = &u.cont
	return u
}

// page is the current page, made on first use.
func (u *UI) page() page {
	id := u.flow.Current()
	p, ok := u.pages[id]
	if !ok {
		if mk, ok := pageMakers[id]; ok {
			p = mk(u)
		} else {
			p = &stubPage{basePage{u}, id}
		}
		u.pages[id] = p
	}
	return p
}

// Layout draws one frame over the whole OS window.
func (u *UI) Layout(gtx layout.Context) layout.Dimensions {
	gtx = guikit.Scaled(gtx, u.scale)
	event.Op(gtx.Ops, u)
	guikit.WindowKeys(gtx, &u.scale, u.prev)
	u.update(gtx)
	if cur := u.flow.Current(); cur != u.shown {
		// A new page slides in; the first one is simply there.
		if u.shown != "" {
			u.pageAt = u.now()
		}
		u.shown = cur
		u.focus = &u.cont
	}
	if u.focus != nil {
		gtx.Execute(key.FocusCmd{Tag: u.focus})
		u.focus = nil
	}

	size := gtx.Constraints.Max
	guikit.FillRect(gtx, image.Rectangle{Max: size}, guikit.ColWindow)
	gtx.Constraints = layout.Exact(size)
	p := u.page()
	u.Frame(gtx, u.steps(), u.slide(p.Body), func(gtx layout.Context) layout.Dimensions {
		lead, trail := p.Footer(u.nav())
		return u.Footer(gtx, lead, nil, trail)
	})
	if !u.pageAt.IsZero() && u.now().Sub(u.pageAt) < guikit.SlideTime {
		gtx.Execute(op.InvalidateCmd{})
	}
	return layout.Dimensions{Size: size}
}

// update handles the footer's buttons, then the page's own input.
func (u *UI) update(gtx layout.Context) {
	if u.back.Clicked(gtx) {
		u.prev()
	}
	if u.skip.Clicked(gtx) {
		u.skipPage()
	}
	if u.cont.Clicked(gtx) {
		u.next()
	}
	u.page().Update(gtx)
}

// steps is the sidebar: the pages of the flow, the current one lit.
func (u *UI) steps() guikit.Steps {
	var names []string
	for _, p := range u.flow.Pages() {
		names = append(names, PageTitle(p))
	}
	return guikit.Steps{Title: "Isotope", Names: names, Current: u.flow.Index(), Note: u.version}
}

// slide is w sliding in as the page changes.
func (u *UI) slide(w layout.Widget) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions { return guikit.Slide(gtx, u.pageAt, u.now(), w) }
}

// nav are the footer buttons for the current page.
func (u *UI) nav() nav {
	n := nav{
		Skip:     &guikit.Action{Click: &u.skip, Kind: guikit.ButtonGhost, Label: "Skip"},
		Continue: &guikit.Action{Click: &u.cont, Kind: guikit.ButtonPrimary, Label: "Continue"},
	}
	if u.flow.Index() == len(u.flow.Pages())-1 {
		n.Continue.Label = "Done"
	}
	if u.flow.Index() > 0 {
		n.Back = &guikit.Action{Click: &u.back, Kind: guikit.ButtonSecondary, Label: "Go back"}
	}
	return n
}

// next is Continue.
func (u *UI) next() {
	u.page().Continue()
	if u.flow.Done() {
		u.finish(u.endStamp())
	}
}

// prev is Go back and Esc.
func (u *UI) prev() { u.flow.Back() }

// skipPage is Skip: the page's setting is left as it was.
func (u *UI) skipPage() {
	u.flow.Skip()
	if u.flow.Done() {
		u.finish(u.endStamp())
	}
}

// goBackTo goes back to an earlier page p, if the flow has it.
func (u *UI) goBackTo(p PageID) {
	for u.flow.Current() != p && u.flow.Index() > 0 {
		u.flow.Back()
	}
}

// endStamp is the stamp of a flow that reached its end.
func (u *UI) endStamp() Stamp {
	if u.appsPending {
		return StampAppsPending
	}
	return StampDone
}

// finish ends the flow once, with stamp s.
func (u *UI) finish(s Stamp) {
	if u.finished {
		return
	}
	u.finished = true
	if u.hooks.Finish != nil {
		u.hooks.Finish(s)
	}
}

// skipSetup is "Skip setup" on the Welcome page.
func (u *UI) skipSetup() { u.finish(StampSkipped) }

// Closed is the window closing. Gio cannot keep it open, so closing before
// Finish is a skip.
func (u *UI) Closed() { u.finish(StampSkipped) }

// scrolled lays w in the page's padding, in a list that scrolls when w is
// taller than the column (a large scale, a small screen). w gets at least
// the column's height, so it can centre itself.
func (u *UI) scrolled(gtx layout.Context, l *widget.List, w layout.Widget) layout.Dimensions {
	l.Axis = layout.Vertical
	h := gtx.Constraints.Max.Y
	return guikit.Scrolled(gtx, l, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.Y = h
		return guikit.Pad(gtx, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			return w(gtx)
		})
	})
}

// basePage is the usual page: Go back, Skip and Continue, where Continue
// moves on without applying anything.
type basePage struct{ u *UI }

func (b basePage) Footer(n nav) (leading, trailing []guikit.Action) {
	if n.Back != nil {
		leading = append(leading, *n.Back)
	}
	return leading, []guikit.Action{*n.Skip, *n.Continue}
}

func (b basePage) Update(layout.Context) {}

func (b basePage) Continue() { b.u.flow.Next() }

// stubPage stands for a page whose file is not written yet.
type stubPage struct {
	basePage
	id PageID
}

func (s *stubPage) Body(gtx layout.Context) layout.Dimensions {
	return guikit.Pad(gtx, s.u.Heading(PageTitle(s.id), "This page is not available yet."))
}
