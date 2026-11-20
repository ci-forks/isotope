package main

import (
	"image"

	"gioui.org/layout"
	"gioui.org/widget"

	"github.com/kairos-community/isotope/guikit"
)

func init() {
	pageMakers[PAppearance] = func(u *UI) page {
		return &appearancePage{basePage: basePage{u}, accents: Accents(), pick: -1}
	}
}

// lightNoteTiling is the light card's line where only the apps follow the
// choice: the bar, terminal, notifications and menus stay dark.
const lightNoteTiling = "Apps use a light theme; the bar and menus stay dark"

// appearancePage is dark or light for the apps and, on COSMIC, the accent
// colour and the frosted effect. Continue writes the choice.
type appearancePage struct {
	basePage
	list widget.List

	started      bool
	dark         bool
	darkC, light widget.Clickable
	accents      []Accent
	swatches     []widget.Clickable
	pick         int // index in accents, -1 for COSMIC's own
	frosted      bool
	frostedC     widget.Clickable
	frostedSet   bool // the user touched the toggle
	err          string
}

// start reads what is set now, once.
func (p *appearancePage) start() {
	if p.started {
		return
	}
	p.started = true
	p.swatches = make([]widget.Clickable, len(p.accents))
	p.dark = ReadColorScheme(p.u.hooks.Runner, p.u.hooks.Home, p.u.desktop)
}

func (p *appearancePage) Update(gtx layout.Context) {
	p.start()
	if p.darkC.Clicked(gtx) {
		p.dark = true
	}
	if p.light.Clicked(gtx) {
		p.dark = false
	}
	if p.u.desktop != Cosmic {
		return
	}
	for i := range p.swatches {
		if p.swatches[i].Clicked(gtx) {
			p.pick = i
		}
	}
	if p.frostedC.Clicked(gtx) {
		p.frosted, p.frostedSet = !p.frosted, true
	}
}

// apply writes the choice. A failure is shown on the page and the flow
// stays, so the user can try again or skip.
func (p *appearancePage) apply() error {
	h := p.u.hooks
	if h.Runner == nil {
		return nil
	}
	if err := SetColorScheme(h.Runner, h.Home, p.u.desktop, p.dark); err != nil {
		return err
	}
	if p.u.desktop != Cosmic {
		return nil
	}
	if p.pick >= 0 {
		if err := SetCosmicAccent(h.Home, p.accents[p.pick]); err != nil {
			return err
		}
	}
	if p.frostedSet {
		return SetCosmicFrosted(h.Home, p.frosted)
	}
	return nil
}

func (p *appearancePage) Continue() {
	if err := p.apply(); err != nil {
		p.err = "Could not change the appearance: " + err.Error()
		return
	}
	p.err = ""
	p.u.flow.Apply()
	p.u.flow.Next()
}

func (p *appearancePage) Body(gtx layout.Context) layout.Dimensions {
	u := p.u
	cosmic := u.desktop == Cosmic
	lightNote := lightNoteTiling
	if cosmic {
		lightNote = "Apps and the desktop use a light theme"
	}
	sub := "Dark is Isotope's look. You can change it later in the Isotope menu."
	return u.scrolled(gtx, &p.list, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Max.X = min(gtx.Constraints.Max.X, guikit.Dp(gtx, 620))
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		kids := []layout.FlexChild{
			layout.Rigid(u.Heading("Appearance", sub)),
			guikit.VGap(22),
		}
		if p.err != "" {
			kids = append(kids, layout.Rigid(func(gtx layout.Context) layout.Dimensions { return u.Warning(gtx, p.err) }), guikit.VGap(14))
		}
		kids = append(kids,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{}.Layout(gtx,
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return u.ChoiceCard(gtx, &p.darkC, p.dark, "Dark", "Easy on the eyes, the Isotope look")
					}),
					layout.Rigid(layout.Spacer{Width: 14}.Layout),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return u.ChoiceCard(gtx, &p.light, !p.dark, "Light", lightNote)
					}))
			}))
		if cosmic {
			kids = append(kids,
				guikit.VGap(26),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return u.Label(gtx, guikit.StBody600, guikit.ColText, "Accent colour")
				}),
				guikit.VGap(10),
				layout.Rigid(p.swatchRow),
				guikit.VGap(22),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return u.CheckCard(gtx, &p.frostedC, p.frosted, "Frosted glass",
						"Blur what is behind the panel, the dock, the menus and the windows.")
				}))
		}
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, kids...)
	})
}

// swatchRow is one round swatch per accent, the picked one ringed.
func (p *appearancePage) swatchRow(gtx layout.Context) layout.Dimensions {
	size := guikit.Dp(gtx, 34)
	var kids []layout.FlexChild
	for i := range p.accents {
		i := i
		kids = append(kids, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Right: 12}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return p.swatches[i].Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					r := image.Rect(0, 0, size, size)
					if p.pick == i {
						guikit.StrokeRRect(gtx, r, size/2, guikit.Fdp(gtx, 3), guikit.ColText)
						guikit.FillRRect(gtx, r.Inset(guikit.Dp(gtx, 5)), size/2, hexColor(p.accents[i].Hex))
					} else {
						guikit.FillRRect(gtx, r, size/2, hexColor(p.accents[i].Hex))
					}
					return layout.Dimensions{Size: r.Max}
				})
			})
		}))
	}
	return layout.Flex{}.Layout(gtx, kids...)
}
