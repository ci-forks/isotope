package main

import (
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/widget"

	"github.com/kairos-community/isotope/guikit"
)

func init() { pageMakers[PWelcome] = func(u *UI) page { return &welcomePage{basePage: basePage{u}} } }

// welcomePage is the first page: the mark, the edition and version, and
// how to set the machine up. "Restore from a backup" joins "Set up as new"
// with the Restore plan.
type welcomePage struct {
	basePage
	list        widget.List
	asNew, quit widget.Clickable
}

func (p *welcomePage) Footer(n nav) (leading, trailing []guikit.Action) {
	return []guikit.Action{{Click: &p.quit, Kind: guikit.ButtonGhost, Label: "Skip setup"}},
		[]guikit.Action{*n.Continue}
}

func (p *welcomePage) Update(gtx layout.Context) {
	if p.quit.Clicked(gtx) {
		p.u.skipSetup()
	}
	if p.asNew.Clicked(gtx) {
		// The only choice: a click on it is a Continue.
		p.u.next()
	}
}

func (p *welcomePage) Body(gtx layout.Context) layout.Dimensions {
	u := p.u
	return u.scrolled(gtx, &p.list, func(gtx layout.Context) layout.Dimensions {
		// A short column (a large scale, a small screen) gets a smaller
		// mark, and a very short one none (the sidebar has it), so the
		// choice stays in view.
		glow, mark := float32(150), float32(110)
		switch h := gtx.Constraints.Min.Y; {
		case h < guikit.Dp(gtx, 420):
			glow = 0
		case h < guikit.Dp(gtx, 560):
			glow, mark = 96, 70
		}
		return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Max.X = min(gtx.Constraints.Max.X, guikit.Dp(gtx, 480))
			return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if glow == 0 {
						return layout.Dimensions{}
					}
					guikit.Glow(gtx, glow, guikit.ColAccent, 0.14)
					o := op.Offset(guikit.Pt(gtx, (glow-mark)/2, (glow-mark)/2)).Push(gtx.Ops)
					guikit.Mark(gtx, mark, false, 1, 0)
					o.Pop()
					return layout.Dimensions{Size: guikit.Pt(gtx, glow, glow)}
				}),
				guikit.VGap(14),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return u.Label(gtx, guikit.StHero, guikit.ColText, "Welcome to Isotope")
				}),
				guikit.VGap(8),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return u.Label(gtx, guikit.StBody, guikit.ColDim, u.version)
				}),
				guikit.VGap(20),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return u.Centered(gtx, guikit.StLead, guikit.ColText2, 1.55,
						"A few choices make this computer yours: the language, the network, the look and the apps. Each step can be skipped, and you can come back to this window later from the Isotope menu.")
				}),
				guikit.VGap(26),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Max.X = min(gtx.Constraints.Max.X, guikit.Dp(gtx, 420))
					return u.ChoiceCard(gtx, &p.asNew, true, "Set up as new", "Choose the settings step by step")
				}),
			)
		})
	})
}
