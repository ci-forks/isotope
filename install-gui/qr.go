package main

import (
	"image"

	"gioui.org/layout"
	"rsc.io/qr"
)

// qrCode draws text as a QR code (error correction M) in a white square of
// size dp with the standard four-module quiet zone, for a phone camera.
func qrCode(gtx layout.Context, text string, size float32) layout.Dimensions {
	px := dp(gtx, size)
	code, err := qr.Encode(text, qr.M)
	if err != nil {
		return layout.Dimensions{}
	}
	n := code.Size + 8
	mod := px / n
	if mod < 1 {
		mod = 1
	}
	side := mod * n
	fillRRect(gtx, image.Rect(0, 0, side, side), dp(gtx, 6), rgb(0xffffff))
	for y := 0; y < code.Size; y++ {
		for x := 0; x < code.Size; x++ {
			if code.Black(x, y) {
				x0, y0 := (x+4)*mod, (y+4)*mod
				fillRect(gtx, image.Rect(x0, y0, x0+mod, y0+mod), rgb(0x000000))
			}
		}
	}
	return layout.Dimensions{Size: image.Pt(side, side)}
}

// qrGlyph is the button's icon: three finder squares and a few modules.
func qrGlyph(gtx layout.Context, size float32) layout.Dimensions {
	k := fdp(gtx, size) / 16
	sq := func(x, y float32) {
		r := image.Rect(int(x*k), int(y*k), int((x+5)*k+0.5), int((y+5)*k+0.5))
		strokeRRect(gtx, r, 0, k, colText2)
	}
	sq(2, 2)
	sq(9, 2)
	sq(2, 9)
	fillRect(gtx, image.Rect(int(9*k), int(9*k), int(11*k), int(11*k)), colText2)
	fillRect(gtx, image.Rect(int(12*k), int(12*k), int(14*k), int(14*k)), colText2)
	s := int(16*k + 0.5)
	return layout.Dimensions{Size: image.Pt(s, s)}
}
