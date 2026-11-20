package main

// PageID names one page of the welcome flow.
type PageID string

const (
	PWelcome       PageID = "welcome"
	PLanguage      PageID = "language"
	PNetwork       PageID = "network"
	PAppearance    PageID = "appearance"
	PLayout        PageID = "layout"
	PWorkflow      PageID = "workflow"
	PLauncher      PageID = "launcher"
	PAccessibility PageID = "accessibility"
	PApps          PageID = "apps"
	PShortcuts     PageID = "shortcuts"
	PReady         PageID = "ready"
)

// Pages returns the pages for a desktop, in order. Layout, Workflow and
// Launcher exist only on COSMIC.
func Pages(d Desktop) []PageID {
	p := []PageID{PWelcome, PLanguage, PNetwork, PAppearance}
	if d == Cosmic {
		p = append(p, PLayout, PWorkflow, PLauncher)
	}
	return append(p, PAccessibility, PApps, PShortcuts, PReady)
}

// PageTitle is the sidebar name of a page.
func PageTitle(p PageID) string {
	switch p {
	case PWelcome:
		return "Welcome"
	case PLanguage:
		return "Language"
	case PNetwork:
		return "Network"
	case PAppearance:
		return "Appearance"
	case PLayout:
		return "Layout"
	case PWorkflow:
		return "Workflow"
	case PLauncher:
		return "Launcher"
	case PAccessibility:
		return "Accessibility"
	case PApps:
		return "Apps"
	case PShortcuts:
		return "Shortcuts"
	case PReady:
		return "Ready"
	}
	return string(p)
}

// Outcome is what the Ready page shows.
type Outcome struct {
	Applied, Skipped []PageID
	LanguageChanged  string
	AppsInstalled    []string
	AppsFailed       []string
}

// Flow is the user's position in the pages.
type Flow struct {
	pages   []PageID
	idx     int
	done    bool
	Outcome Outcome
}

// NewFlow starts the full flow for a desktop.
func NewFlow(d Desktop) *Flow { return &Flow{pages: Pages(d)} }

// Single is a flow of one page, for `isotope welcome apps`.
func Single(p PageID) *Flow { return &Flow{pages: []PageID{p}} }

// Current is the page being shown.
func (f *Flow) Current() PageID { return f.pages[f.idx] }

// Index is the position of the current page.
func (f *Flow) Index() int { return f.idx }

// Pages lists the pages of this flow.
func (f *Flow) Pages() []PageID { return f.pages }

// Done reports that Next was called on the last page.
func (f *Flow) Done() bool { return f.done }

// Next advances; on the last page it ends the flow.
func (f *Flow) Next() {
	if f.idx+1 >= len(f.pages) {
		f.done = true
		return
	}
	f.idx++
}

// Back goes to the previous page and stays at the first.
func (f *Flow) Back() {
	if f.idx > 0 {
		f.idx--
	}
}

// Skip marks the current page skipped, then moves on.
func (f *Flow) Skip() {
	f.Outcome.Skipped = append(f.Outcome.Skipped, f.Current())
	f.Next()
}

// Apply marks the current page applied.
func (f *Flow) Apply() {
	f.Outcome.Applied = append(f.Outcome.Applied, f.Current())
}
