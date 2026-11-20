package guikit

import "image/color"

// The colours of the graphical installer design note's "Visual system"
// table, which are theme/palette.toml's night palette, the same as the
// splash and GRUB.
var (
	ColDesk1     = RGB(0x2c3a1c) // the glow at the top of the desktop
	ColDesk2     = RGB(0x15141c)
	ColDesk3     = RGB(0x0e0d13)
	ColWindow    = RGB(0x1f1e28)
	ColSidebar   = RGB(0x191820)
	ColFooter    = RGB(0x1c1b24)
	ColField     = RGB(0x2a2935)
	ColAccent    = RGB(0x8fd12e)
	ColOnAccent  = RGB(0x15141c)
	ColGlow      = RGB(0xc6f36b)
	ColCheck     = RGB(0x4e9a06)
	ColText      = RGB(0xf2f1f5)
	ColText2     = RGB(0xcbc8d6)
	ColDim       = RGB(0x8f8ba3)
	ColRing      = RGB(0x5c596a)
	ColError     = RGB(0xf7768e)
	ColErrorText = RGB(0xf7a3b3)
	ColDisabled  = RGB(0x3c3a49)
	ColKeyBox    = RGB(0x15141c)
	ColWarn      = RGB(0xe0af68) // the palette's yellow: a warning, not an error
)

// FocusRing is the accent ring of the mocks, shown around a focused input.
var FocusRing = Over(ColWindow, ColAccent, 0.18)

// FocusRingStrong marks the focused button or card. It is light, so it
// shows around the lime button too. The inputs' faint ring sits next to a
// solid accent border; a button has none, and at 18% the ring did not show
// on the footer at all.
var FocusRingStrong = Over(ColWindow, ColText, 0.7)

// RGB is the opaque colour 0xRRGGBB.
func RGB(v uint32) color.NRGBA {
	return color.NRGBA{R: uint8(v >> 16), G: uint8(v >> 8), B: uint8(v), A: 0xff}
}

// Alpha is c at opacity a (0..1), as the mocks' rgba() values.
func Alpha(c color.NRGBA, a float32) color.NRGBA {
	c.A = uint8(a*255 + 0.5)
	return c
}

// Over is c at opacity a laid on bg, mixed in sRGB as a browser does, so the
// mocks' rgba() values come out as drawn there. Gio blends in linear light,
// which makes a faint overlay much brighter, so translucent fills are mixed
// here into opaque colours instead.
func Over(bg, c color.NRGBA, a float32) color.NRGBA {
	m := func(x, y uint8) uint8 { return uint8(float32(x)*(1-a) + float32(y)*a + 0.5) }
	return color.NRGBA{R: m(bg.R, c.R), G: m(bg.G, c.G), B: m(bg.B, c.B), A: 0xff}
}
