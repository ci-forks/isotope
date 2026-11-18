package main

import (
	"image"
	"image/color"

	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget"
)

type btnKind int

const (
	btnPrimary   btnKind = iota // accent: Continue, Restart
	btnSecondary                // Go back
	btnGhost                    // Shut down, Show details
	btnDanger                   // Erase and install
)

// focusRing is the accent ring of the mocks, shown on the focused control.
var focusRing = over(colWindow, colAccent, 0.18)

// button draws a 40 px button of the mocks. A disabled button is drawn
// greyed and ignores clicks and keys.
func (u *UI) button(gtx layout.Context, c *widget.Clickable, kind btnKind, txt string, enabled bool) layout.Dimensions {
	if !enabled {
		gtx = gtx.Disabled()
	}
	padX := float32(22)
	st, fg := stButton, colOnAccent
	b := box{radius: 8, fill: colAccent}
	switch {
	case !enabled:
		b.fill, fg = colDisabled, colDim
	case kind == btnSecondary:
		padX, st, fg = 18, stButton500, colText
		b.fill, b.border, b.bw = colField, over(colFooter, colText, 0.1), 1
	case kind == btnGhost:
		padX, st, fg = 16, stButton500, colDim
		b.fill = color.NRGBA{}
	case kind == btnDanger:
		b.fill = colError
	}
	if enabled && gtx.Focused(c) {
		b.ring, b.ringW = focusRing, 3
		if kind == btnGhost {
			fg = colText
		}
	}
	return c.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		if enabled {
			pointer.CursorPointer.Add(gtx.Ops)
		}
		gtx.Constraints.Min.Y = dp(gtx, 40)
		return b.layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Left: unit.Dp(padX), Right: unit.Dp(padX)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min.X = 0
				return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return u.label(gtx, st, fg, txt)
				})
			})
		})
	})
}

// field is a labelled text input with an error line under it.
type field struct {
	ed   widget.Editor
	err  string
	mask bool
}

func newField(submit bool) *field {
	f := &field{}
	f.ed.SingleLine = true
	f.ed.Submit = submit
	return f
}

func newSecret() *field {
	f := newField(true)
	f.ed.Mask = '•'
	f.mask = true
	return f
}

// input draws the editor in its 38 px box. prefix, when set, is drawn in the
// box before the text (the "github.com/" of the SSH key field).
func (u *UI) input(gtx layout.Context, f *field, s style, height float32, prefix string, trailing layout.Widget) layout.Dimensions {
	focused := gtx.Focused(&f.ed)
	b := box{fill: colField, border: over(colField, colText, 0.08), bw: 1, radius: 8}
	if focused {
		b.border, b.ring, b.ringW = colAccent, focusRing, 3
	}
	if f.err != "" {
		b.border = colError
	}
	gtx.Constraints.Min.Y = dp(gtx, height)
	gtx.Constraints.Max.Y = dp(gtx, height)
	return b.layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return layout.Inset{Left: 12, Right: 12}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min = gtx.Constraints.Max
			var kids []layout.FlexChild
			if prefix != "" {
				kids = append(kids, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.Y = gtx.Constraints.Max.Y
					return layout.W.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						gtx.Constraints.Min.Y = 0
						return u.label(gtx, s, colDim, prefix)
					})
				}))
			}
			kids = append(kids, layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min.Y = gtx.Constraints.Max.Y
				// W keeps the vertical minimum; zero it so the text is
				// centred rather than drawn at the top of the box.
				return layout.W.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min = image.Pt(gtx.Constraints.Max.X, 0)
					return f.ed.Layout(gtx, u.shaper, s.font(), s.size, material(gtx, colText), material(gtx, alpha(colAccent, 0.35)))
				})
			}))
			if trailing != nil {
				kids = append(kids, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.Y = gtx.Constraints.Max.Y
					return layout.E.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						gtx.Constraints.Min.Y = 0
						return trailing(gtx)
					})
				}))
			}
			return layout.Flex{Alignment: layout.Middle}.Layout(gtx, kids...)
		})
	})
}

// labelled stacks a label, the input and a hint or the field's error.
func (u *UI) labelled(gtx layout.Context, title string, f *field, hint string, in layout.Widget) layout.Dimensions {
	kids := []layout.FlexChild{
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return u.label(gtx, stLabel, colText2, title) }),
		layout.Rigid(layout.Spacer{Height: 6}.Layout),
		layout.Rigid(in),
	}
	msg, col := hint, colDim
	if f != nil && f.err != "" {
		msg, col = f.err, colError
	}
	if msg != "" {
		kids = append(kids,
			layout.Rigid(layout.Spacer{Height: 6}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { return u.label(gtx, stTiny, col, msg) }))
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, kids...)
}

// radio is the round marker of the encryption cards.
func radio(gtx layout.Context, on bool) layout.Dimensions {
	s := dp(gtx, 18)
	r := image.Rect(0, 0, s, s)
	if on {
		strokeRRect(gtx, r, s/2, fdp(gtx, 5), colAccent)
	} else {
		strokeRRect(gtx, r, s/2, fdp(gtx, 1.5), colRing)
	}
	return layout.Dimensions{Size: r.Max}
}

// checkbox is the recovery key box: accent filled with a tick when on.
func checkbox(gtx layout.Context, on bool) layout.Dimensions {
	s := dp(gtx, 18)
	r := image.Rect(0, 0, s, s)
	if on {
		fillRRect(gtx, r, dp(gtx, 4), colAccent)
		m := op.Offset(image.Pt(dp(gtx, 1), dp(gtx, 1))).Push(gtx.Ops)
		tick(gtx, 16, colOnAccent)
		m.Pop()
	} else {
		fillRRect(gtx, r, dp(gtx, 4), colField)
		strokeRRect(gtx, r, dp(gtx, 4), fdp(gtx, 1.5), colRing)
	}
	return layout.Dimensions{Size: r.Max}
}

// card is a selectable card: the disk and encryption choices.
func (u *UI) card(gtx layout.Context, c *widget.Clickable, selected bool, pad layout.Inset, w layout.Widget) layout.Dimensions {
	b := box{radius: 12, fill: colSidebar, border: over(colSidebar, colText, 0.08), bw: 1}
	if selected {
		b.fill, b.border, b.bw = over(colWindow, colAccent, 0.08), colAccent, 2
		b.ring, b.ringW = over(colWindow, colAccent, 0.12), 4
	}
	if gtx.Focused(c) {
		b.ring, b.ringW = focusRing, 3
		if !selected {
			b.border = over(colWindow, colAccent, 0.6)
		}
	}
	return c.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		pointer.CursorPointer.Add(gtx.Ops)
		return b.layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return pad.Layout(gtx, w)
		})
	})
}

// panel is the mocks' dark inset box (the SSH key box, the key list).
func panel(gtx layout.Context, radius float32, pad layout.Inset, w layout.Widget) layout.Dimensions {
	b := box{radius: radius, fill: colSidebar, border: over(colSidebar, colText, 0.06), bw: 1}
	return b.layout(gtx, func(gtx layout.Context) layout.Dimensions { return pad.Layout(gtx, w) })
}

// warning is the red box with an icon and words.
func (u *UI) warning(gtx layout.Context, txt string) layout.Dimensions {
	b := box{radius: 10, fill: over(colWindow, colError, 0.1), border: over(colWindow, colError, 0.35), bw: 1}
	return b.layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return layout.Inset{Top: 12, Bottom: 12, Left: 16, Right: 16}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions { return warnIcon(gtx, 18, colError) }),
				layout.Rigid(layout.Spacer{Width: 12}.Layout),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return u.label(gtx, stBody, colErrorText, txt) }),
			)
		})
	})
}

// link is a text button in the glow colour: the summary's "Change".
func (u *UI) link(gtx layout.Context, c *widget.Clickable, txt string) layout.Dimensions {
	col := colGlow
	if c.Hovered() || gtx.Focused(c) {
		col = colText
	}
	return c.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		pointer.CursorPointer.Add(gtx.Ops)
		dims := u.label(gtx, stSmall, col, txt)
		if gtx.Focused(c) {
			fillRect(gtx, image.Rect(0, dims.Size.Y-dp(gtx, 1), dims.Size.X, dims.Size.Y), colAccent)
		}
		return dims
	})
}
