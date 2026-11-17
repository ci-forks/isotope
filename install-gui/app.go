package main

import (
	"fmt"
	"image"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/text"
	"gioui.org/widget"

	"github.com/kairos-community/isotope/install-ui/core"
)

// gpage is a page of the graphical installer. The text installer's fourteen
// steps (core.Page) fold into seven pages, because a window has room for
// several fields; guiPage maps one to the other.
type gpage int

const (
	pWelcome gpage = iota
	pKeyboard
	pAccount
	pDisk
	pEncrypt
	pRecovery
	pSummary
	pInstalling
	pDone
)

// stepNames is the sidebar's step list.
var stepNames = []string{"Introduction", "Keyboard and time zone", "Account", "Disk", "Encryption", "Summary", "Installation"}

func stepOf(p gpage) int {
	switch p {
	case pWelcome:
		return 0
	case pKeyboard:
		return 1
	case pAccount:
		return 2
	case pDisk:
		return 3
	case pEncrypt, pRecovery:
		return 4
	case pSummary:
		return 5
	}
	return 6
}

func guiPage(p core.Page) gpage {
	switch p {
	case core.PageKeyboard, core.PageTimezone:
		return pKeyboard
	case core.PageHostname, core.PageUser, core.PagePass, core.PageGithub:
		return pAccount
	case core.PageDisk:
		return pDisk
	case core.PageEncrypt, core.PagePassphrase, core.PagePassphraseConfirm, core.PageRecovery:
		return pEncrypt
	case core.PageRecoveryShow, core.PageRecoveryConfirm:
		return pRecovery
	}
	return pSummary
}

// firstCorePage is the core page a GUI page starts at.
func (u *UI) firstCorePage(p gpage) core.Page {
	switch p {
	case pKeyboard:
		return core.PageKeyboard
	case pAccount:
		return core.PageHostname
	case pDisk:
		return core.PageDisk
	case pEncrypt:
		if u.w.TrustedBoot {
			return core.PageRecovery
		}
		return core.PageEncrypt
	case pRecovery:
		return core.PageRecoveryShow
	}
	return core.PageReview
}

// hooks are what the UI does to the world; the screenshot tests replace
// them.
type hooks struct {
	// confirm hands the reviewed choice to the install.
	confirm func(core.Choice)
	// relaunch saves s and restarts the window with the layout s names.
	relaunch func(s core.GUIState)
	poweroff func()
	reboot   func()
	// started is called once the first frame is drawn.
	started func()
}

// UI is the graphical installer: the window's state and the wizard it
// drives. Every rule is core's: the UI collects a page's fields, calls the
// core.Wizard methods for that page, shows their errors and follows Page.
type UI struct {
	shaper *text.Shaper
	w      *core.Wizard
	page   gpage
	pageAt time.Time
	now    func() time.Time
	hooks  hooks

	// version is the sidebar's footer, "Isotope v0.0.20 · sway".
	version string
	// runLayout and runVariant are the keymap the compositor was started
	// with; canRelaunch says isotope-install-ui can restart it with another.
	runLayout, runVariant string
	canRelaunch           bool

	focus     event.Tag // focus this on the next frame
	scale     float32   // Ctrl+Plus and Ctrl+Minus
	bg        desktop
	startedOK bool

	back, cont, shutdown widget.Clickable

	kbSearch, kbTry *field
	kbList          widget.List
	kbRows          []widget.Clickable
	kbSel           int
	kbShown         []int
	kbApplied       bool
	tzField         *field
	tzList          widget.List
	tzRows          []widget.Clickable
	tzSel           int
	tzShown         []int

	host, user, pw, pw2, gh *field
	acctList                widget.List
	encList                 widget.List

	diskSel   int
	diskCards []widget.Clickable
	diskList  widget.List
	rescan    widget.Clickable

	encOn, recKey  bool
	encNo, encYes  widget.Clickable
	recBox         widget.Clickable
	pp, pp2        *field
	encErr         string
	rk             *field
	change         [6]widget.Clickable
	summaryChanges []gpage

	install installState
	details widget.Clickable
	logList widget.List
	restart widget.Clickable
}

// installState is what the installing page shows, fed by core.Install.
type installState struct {
	step     int
	lines    []string
	failed   bool
	err      error
	finished bool
	encrypt  bool
	user     string
	show     bool
	events   chan core.Event
}

const logKeep = 200

func newUI(w *core.Wizard, shaper *text.Shaper, h hooks) *UI {
	u := &UI{shaper: shaper, w: w, hooks: h, now: time.Now, scale: 1}
	u.kbSearch = newField(true)
	u.kbTry = newField(true)
	u.tzField = newField(true)
	u.host = newField(true)
	u.user = newField(true)
	u.pw = newSecret()
	u.pw2 = newSecret()
	u.gh = newField(true)
	u.pp = newSecret()
	u.pp2 = newSecret()
	u.rk = newField(true)
	u.rk.ed.MaxLen = 80
	u.kbList.Axis = layout.Vertical
	u.tzList.Axis = layout.Vertical
	u.logList.Axis = layout.Vertical
	u.logList.ScrollToEnd = true
	u.diskList.Axis = layout.Vertical
	u.acctList.Axis = layout.Vertical
	u.encList.Axis = layout.Vertical
	u.install.events = make(chan core.Event, 256)
	u.runLayout = core.DefaultLayout
	if w != nil {
		u.setWizard(w)
	}
	return u
}

// setWizard starts the UI on w, the wizard core.RunWizard made for this
// machine.
func (u *UI) setWizard(w *core.Wizard) {
	u.w = w
	u.kbSel = w.KeyboardIndex()
	u.tzSel = w.TimezoneIndex()
	if u.tzSel >= 0 {
		u.tzField.ed.SetText(w.Zones[u.tzSel])
	}
	u.host.ed.SetText(w.HostnameValue())
	u.kbRows = make([]widget.Clickable, len(w.Keyboards))
	u.tzRows = make([]widget.Clickable, len(w.Zones))
	u.diskCards = make([]widget.Clickable, len(w.Disks))
	u.scrollToSelection()
	// Continue has the focus, so Enter goes on from the first page.
	u.refocus()
}

func (u *UI) scrollToSelection() {
	u.filterKeyboards()
	u.filterZones()
	for i, k := range u.kbShown {
		if k == u.kbSel {
			u.kbList.Position.First = max(0, i-2)
		}
	}
	for i, z := range u.tzShown {
		if z == u.tzSel {
			u.tzList.Position.First = max(0, i-2)
		}
	}
}

// goTo shows p; the page slides in.
func (u *UI) goTo(p gpage) {
	if p != u.page {
		u.pageAt = u.now()
	}
	u.page = p
	u.refocus()
}

// refocus gives the page's first control the keyboard focus: on a new page,
// and when the window gets the keyboard (Gio drops a focus set before that).
func (u *UI) refocus() {
	switch u.page {
	case pWelcome, pDisk:
		u.focus = &u.cont
	case pKeyboard:
		u.focus = &u.kbSearch.ed
		if u.kbApplied {
			// A layout was just applied: try it.
			u.focus = &u.kbTry.ed
		}
	case pAccount:
		u.focus = &u.user.ed
	case pEncrypt:
		switch {
		case u.encOn && !u.w.TrustedBoot:
			u.focus = &u.pp.ed
		case u.encOn || u.recKey:
			u.focus = &u.encYes
		default:
			u.focus = &u.encNo
		}
	case pRecovery:
		u.focus = &u.rk.ed
	case pSummary:
		u.focus = &u.back
	case pDone:
		u.focus = &u.restart
	case pInstalling:
		if u.install.failed {
			u.focus = &u.restart
		}
	}
}

// next is Continue.
func (u *UI) next() {
	switch u.page {
	case pWelcome:
		u.goTo(pKeyboard)
	case pKeyboard:
		u.nextKeyboard()
	case pAccount:
		u.nextAccount()
	case pDisk:
		u.nextDisk()
	case pEncrypt:
		u.nextEncrypt()
	case pRecovery:
		u.nextRecovery()
	case pSummary:
		u.hooks.confirm(u.w.Choice)
		u.install.encrypt = u.w.Choice.Encrypt
		u.install.user = u.w.Choice.Username
		u.goTo(pInstalling)
	case pDone:
		u.hooks.reboot()
	}
}

// prev is Go back and Esc: the core page before this GUI page.
func (u *UI) prev() {
	switch u.page {
	case pWelcome, pInstalling, pDone:
		return
	case pKeyboard:
		u.goTo(pWelcome)
		return
	}
	cur := u.page
	u.w.Page = u.firstCorePage(cur)
	for guiPage(u.w.Page) == cur {
		if !u.w.Back() {
			u.goTo(pWelcome)
			return
		}
	}
	u.goTo(guiPage(u.w.Page))
}

// --- keyboard and time zone -------------------------------------------------

func (u *UI) filterKeyboards() {
	q := strings.ToLower(strings.TrimSpace(u.kbSearch.ed.Text()))
	u.kbShown = u.kbShown[:0]
	for i, k := range u.w.Keyboards {
		if q == "" || strings.Contains(strings.ToLower(k.Description), q) || strings.Contains(strings.ToLower(k.Code()), q) {
			u.kbShown = append(u.kbShown, i)
		}
	}
}

func zoneMatch(zone, q string) bool {
	z := strings.ToLower(strings.ReplaceAll(zone, "_", " "))
	return strings.Contains(z, strings.ReplaceAll(q, "_", " "))
}

func (u *UI) filterZones() {
	q := strings.ToLower(strings.TrimSpace(u.tzField.ed.Text()))
	if u.tzSel >= 0 && strings.EqualFold(q, u.w.Zones[u.tzSel]) {
		q = ""
	}
	u.tzShown = u.tzShown[:0]
	for i, z := range u.w.Zones {
		if q == "" || zoneMatch(z, q) {
			u.tzShown = append(u.tzShown, i)
		}
	}
}

// pickKeyboard selects Keyboards[i] and applies it at once: a layout the
// compositor was not started with restarts it (see relaunch.go).
func (u *UI) pickKeyboard(i int) {
	u.kbSel = i
	if u.needsRelaunch() {
		u.hooks.relaunch(u.saveState(pKeyboard))
		return
	}
	u.kbApplied = true
	u.focus = &u.kbTry.ed
}

func (u *UI) needsRelaunch() bool {
	if !u.canRelaunch || u.kbSel < 0 {
		return false
	}
	k := u.w.Keyboards[u.kbSel]
	return k.Layout != u.runLayout || k.Variant != u.runVariant
}

func (u *UI) pickZone(i int) {
	u.tzSel = i
	u.tzField.ed.SetText(u.w.Zones[i])
	u.tzField.ed.SetCaret(len(u.w.Zones[i]), len(u.w.Zones[i]))
	u.tzField.err = ""
	u.filterZones()
}

func (u *UI) nextKeyboard() {
	u.tzField.err = ""
	if u.kbSel < 0 {
		return
	}
	if u.tzSel < 0 || u.tzField.ed.Text() != u.w.Zones[u.tzSel] {
		// The field takes a zone from the list; Enter picks the first
		// match, so a typed city is one key away.
		if len(u.tzShown) == 1 || (len(u.tzShown) > 0 && u.tzField.ed.Text() != "") {
			u.pickZone(u.tzShown[0])
		} else {
			u.tzField.err = "Pick a time zone from the list."
			u.focus = &u.tzField.ed
			return
		}
	}
	u.w.SetKeyboard(u.kbSel)
	u.w.SetTimezone(u.tzSel)
	if u.needsRelaunch() {
		u.hooks.relaunch(u.saveState(pAccount))
		return
	}
	u.goTo(pAccount)
}

// --- account ----------------------------------------------------------------

// sentence is core's error text as a sentence for the window: the words are
// the text installer's, with a capital and a full stop.
func sentence(err error) string {
	s := err.Error()
	r, n := utf8.DecodeRuneInString(s)
	s = string(unicode.ToUpper(r)) + s[n:]
	if !strings.HasSuffix(s, ".") {
		s += "."
	}
	return s
}

func (u *UI) passwordsMatch() bool { return u.pw.ed.Text() == u.pw2.ed.Text() }

func (u *UI) nextAccount() {
	for _, f := range []*field{u.host, u.user, u.pw, u.pw2, u.gh} {
		f.err = ""
	}
	w := u.w
	w.SetHostname(u.host.ed.Text())
	var first *field
	fail := func(f *field, err error) {
		f.err = sentence(err)
		if first == nil {
			first = f
		}
	}
	if err := w.SetUsername(u.user.ed.Text()); err != nil {
		fail(u.user, err)
	}
	if err := w.SetPassword(u.pw.ed.Text()); err != nil {
		fail(u.pw, err)
	} else if !u.passwordsMatch() {
		fail(u.pw2, fmt.Errorf("the passwords do not match"))
	}
	if err := w.SetGithub(u.gh.ed.Text()); err != nil && len(w.Disks) > 0 {
		fail(u.gh, err)
	}
	if first != nil {
		u.focus = &first.ed
		return
	}
	// With no disk, SetGithub stays: the disk page says so and rescans.
	w.Page = core.PageDisk
	if u.diskSel >= len(w.Disks) {
		u.diskSel = 0
	}
	u.goTo(pDisk)
}

// --- disk -------------------------------------------------------------------

// sysBlock is where Rescan looks for disks.
var sysBlock = "/sys/block"

func (u *UI) rescanDisks() {
	disks, err := core.ListDisks(sysBlock)
	if err != nil {
		disks = nil
	}
	u.w.Disks = disks
	u.diskCards = make([]widget.Clickable, len(disks))
	u.diskSel = 0
}

func (u *UI) nextDisk() {
	if len(u.w.Disks) == 0 {
		return
	}
	u.w.SetDisk(u.diskSel)
	u.goTo(guiPage(u.w.Page))
}

// sizeText is a disk size as the mocks write it: "1 TB", "64 GB".
func sizeText(gb int) string {
	if gb >= 1000 {
		tb := float64(gb) / 1000
		if tb == float64(int(tb)) || tb >= 10 {
			return fmt.Sprintf("%.0f TB", tb)
		}
		return fmt.Sprintf("%.1f TB", tb)
	}
	return fmt.Sprintf("%d GB", gb)
}

// --- encryption and the recovery key ----------------------------------------

func (u *UI) nextEncrypt() {
	u.pp.err, u.pp2.err, u.encErr = "", "", ""
	w := u.w
	if !w.TrustedBoot {
		w.SetEncrypt(u.encOn)
		if u.encOn {
			if err := w.SetPassphrase(u.pp.ed.Text()); err != nil {
				u.pp.err = sentence(err)
				u.focus = &u.pp.ed
				return
			}
			if err := w.ConfirmPassphrase(u.pp2.ed.Text()); err != nil {
				u.pp2.err = sentence(err)
				u.focus = &u.pp2.ed
				return
			}
		}
	}
	if w.TrustedBoot || u.encOn {
		if err := w.SetRecoveryKey(u.recKey); err != nil {
			u.encErr = sentence(err)
			return
		}
	}
	if w.Page == core.PageRecoveryShow {
		// The key and the check share one page: the group to type back
		// is picked as the page opens.
		w.KeySaved()
		u.rk.ed.SetText("")
		u.rk.err = ""
		u.goTo(pRecovery)
		return
	}
	u.goTo(pSummary)
}

// recoveryConfirmed says whether the typed group is right, without moving
// the wizard: Continue stays disabled until it is.
func (u *UI) recoveryConfirmed() bool {
	c := *u.w
	return c.ConfirmRecoveryKey(u.rk.ed.Text()) == nil
}

func (u *UI) nextRecovery() {
	if err := u.w.ConfirmRecoveryKey(u.rk.ed.Text()); err != nil {
		u.rk.err = sentence(err)
		return
	}
	u.goTo(pSummary)
}

// --- install ----------------------------------------------------------------

// drain applies the install's events that arrived since the last frame.
func (u *UI) drain() {
	for {
		select {
		case e := <-u.install.events:
			u.apply(e)
		default:
			return
		}
	}
}

func (u *UI) apply(e core.Event) {
	s := &u.install
	if u.page < pInstalling {
		// An install that needed no questions (a seeded config).
		u.goTo(pInstalling)
	}
	switch e.Kind {
	case core.EventLine:
		s.lines = append(s.lines, e.Line)
		if len(s.lines) > logKeep {
			s.lines = s.lines[len(s.lines)-logKeep:]
		}
	case core.EventStep:
		s.step = e.Step
	case core.EventFinished:
		s.finished = true
		u.goTo(pDone)
	case core.EventFailed:
		s.failed, s.err = true, e.Err
		u.page = pInstalling
		u.focus = &u.restart
	}
}

// visibleSteps are the install stages the checklist shows: not the last
// ("Complete"), and the passphrase check only for an encrypted install.
func (u *UI) visibleSteps() []int {
	var out []int
	for i, s := range core.Steps[:len(core.Steps)-1] {
		if strings.Contains(s.Match, "checking the disk passphrase") && !u.install.encrypt {
			continue
		}
		out = append(out, i)
	}
	return out
}

// --- keys -------------------------------------------------------------------

// keys handles the window-wide keys: Esc is Go back, Ctrl+Plus and
// Ctrl+Minus scale the window.
func (u *UI) keys(gtx layout.Context) {
	for {
		e, ok := gtx.Event(
			key.Filter{Name: key.NameEscape},
			key.Filter{Name: "+", Required: key.ModShortcut, Optional: key.ModShift},
			key.Filter{Name: "=", Required: key.ModShortcut},
			key.Filter{Name: "-", Required: key.ModShortcut},
			key.Filter{Name: "0", Required: key.ModShortcut},
		)
		if !ok {
			return
		}
		k, ok := e.(key.Event)
		if !ok || k.State != key.Press {
			continue
		}
		switch k.Name {
		case key.NameEscape:
			u.prev()
		case "+", "=":
			u.scale = min(2, u.scale+0.25)
		case "-":
			u.scale = max(1, u.scale-0.25)
		case "0":
			u.scale = 1
		}
	}
}

// update handles the input of the current page before it is drawn.
func (u *UI) update(gtx layout.Context) {
	if u.cont.Clicked(gtx) {
		u.next()
	}
	if u.back.Clicked(gtx) {
		u.prev()
	}
	if u.shutdown.Clicked(gtx) {
		u.hooks.poweroff()
	}
	if u.restart.Clicked(gtx) {
		u.hooks.reboot()
	}
	if u.details.Clicked(gtx) {
		u.install.show = !u.install.show
	}
	submitted := func(f *field) bool {
		hit := false
		for {
			e, ok := f.ed.Update(gtx)
			if !ok {
				return hit
			}
			if _, ok := e.(widget.SubmitEvent); ok {
				hit = true
			}
		}
	}
	switch u.page {
	case pKeyboard:
		before := u.kbSearch.ed.Text()
		if submitted(u.kbSearch) && u.kbSel >= 0 {
			u.pickKeyboard(u.kbSel)
		}
		if u.kbSearch.ed.Text() != before {
			u.filterKeyboards()
			if len(u.kbShown) > 0 {
				u.kbSel = u.kbShown[0]
			}
			u.kbList.Position.First = 0
		}
		for {
			e, ok := gtx.Event(
				key.Filter{Focus: &u.kbSearch.ed, Name: key.NameUpArrow},
				key.Filter{Focus: &u.kbSearch.ed, Name: key.NameDownArrow},
			)
			if !ok {
				break
			}
			if k, ok := e.(key.Event); ok && k.State == key.Press {
				u.kbSel = step(u.kbShown, u.kbSel, k.Name == key.NameDownArrow, &u.kbList)
			}
		}
		for _, i := range u.kbShown {
			if u.kbRows[i].Clicked(gtx) {
				u.pickKeyboard(i)
			}
		}
		if submitted(u.kbTry) {
			u.nextKeyboard()
			return
		}
		beforeTZ := u.tzField.ed.Text()
		if submitted(u.tzField) {
			u.nextKeyboard()
			return
		}
		if u.tzField.ed.Text() != beforeTZ {
			u.filterZones()
			u.tzList.Position.First = 0
		}
		for {
			e, ok := gtx.Event(
				key.Filter{Focus: &u.tzField.ed, Name: key.NameUpArrow},
				key.Filter{Focus: &u.tzField.ed, Name: key.NameDownArrow},
			)
			if !ok {
				break
			}
			if k, ok := e.(key.Event); ok && k.State == key.Press {
				if i := step(u.tzShown, u.tzSel, k.Name == key.NameDownArrow, &u.tzList); i >= 0 {
					u.pickZone(i)
				}
			}
		}
		for _, i := range u.tzShown {
			if u.tzRows[i].Clicked(gtx) {
				u.pickZone(i)
			}
		}
	case pAccount:
		for _, f := range []*field{u.host, u.user, u.pw, u.pw2, u.gh} {
			if submitted(f) {
				u.next()
				return
			}
		}
	case pDisk:
		for i := range u.diskCards {
			if u.diskCards[i].Clicked(gtx) {
				u.diskSel = i
			}
		}
		if u.rescan.Clicked(gtx) {
			u.rescanDisks()
		}
	case pEncrypt:
		if u.encNo.Clicked(gtx) {
			u.encOn = false
			if u.w.TrustedBoot {
				u.recKey = false
			}
		}
		if u.encYes.Clicked(gtx) {
			if u.w.TrustedBoot {
				u.recKey = true
			} else {
				u.encOn = true
				u.focus = &u.pp.ed
			}
		}
		if u.recBox.Clicked(gtx) {
			u.recKey = !u.recKey
		}
		for _, f := range []*field{u.pp, u.pp2} {
			if submitted(f) {
				u.next()
				return
			}
		}
	case pRecovery:
		if submitted(u.rk) && u.recoveryConfirmed() {
			u.next()
			return
		}
	case pSummary:
		for i, p := range u.summaryChanges {
			if u.change[i].Clicked(gtx) {
				u.w.Page = u.firstCorePage(p)
				u.goTo(p)
			}
		}
	}
}

// step moves sel to the next or previous entry of shown, and scrolls the
// list to keep it in view.
func step(shown []int, sel int, down bool, l *widget.List) int {
	if len(shown) == 0 {
		return sel
	}
	pos := -1
	for i, v := range shown {
		if v == sel {
			pos = i
		}
	}
	switch {
	case pos < 0:
		pos = 0
	case down && pos < len(shown)-1:
		pos++
	case !down && pos > 0:
		pos--
	}
	if pos < l.Position.First {
		l.Position.First = pos
		l.Position.Offset = 0
	} else if pos >= l.Position.First+l.Position.Count-1 && l.Position.Count > 0 {
		l.Position.First = pos - l.Position.Count + 2
		l.Position.Offset = 0
	}
	return shown[pos]
}

// pt is a convenience for image.Pt in dp.
func pt(gtx layout.Context, x, y float32) image.Point { return image.Pt(dp(gtx, x), dp(gtx, y)) }
