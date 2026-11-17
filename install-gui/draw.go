package main

import (
	"image"
	"image/color"
	"math"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
)

// dp converts the mocks' CSS pixels, which are logical pixels, to device
// pixels.
func dp(gtx layout.Context, v float32) int { return gtx.Dp(unit.Dp(v)) }

func fdp(gtx layout.Context, v float32) float32 { return float32(gtx.Dp(unit.Dp(v*64))) / 64 }

// fillRect fills r with c.
func fillRect(gtx layout.Context, r image.Rectangle, c color.NRGBA) {
	paint.FillShape(gtx.Ops, c, clip.Rect(r).Op())
}

// fillRRect fills r, with corners of radius rad, with c.
func fillRRect(gtx layout.Context, r image.Rectangle, rad int, c color.NRGBA) {
	paint.FillShape(gtx.Ops, c, clip.UniformRRect(r, rad).Op(gtx.Ops))
}

// strokeRRect draws a border of width w inside r.
func strokeRRect(gtx layout.Context, r image.Rectangle, rad int, w float32, c color.NRGBA) {
	if c.A == 0 || w <= 0 {
		return
	}
	paint.FillShape(gtx.Ops, c, clip.Stroke{Path: rrPath(gtx, r, float32(rad), w/2), Width: w}.Op())
}

// rrPath is the outline of r inset by in, with corner radius rad.
func rrPath(gtx layout.Context, r image.Rectangle, rad, in float32) clip.PathSpec {
	x0, y0 := float32(r.Min.X)+in, float32(r.Min.Y)+in
	x1, y1 := float32(r.Max.X)-in, float32(r.Max.Y)-in
	rad -= in
	if rad < 0 {
		rad = 0
	}
	const k = 0.5523 // cubic approximation of a quarter circle
	var p clip.Path
	p.Begin(gtx.Ops)
	p.MoveTo(f32.Pt(x0+rad, y0))
	p.LineTo(f32.Pt(x1-rad, y0))
	p.CubeTo(f32.Pt(x1-rad+rad*k, y0), f32.Pt(x1, y0+rad-rad*k), f32.Pt(x1, y0+rad))
	p.LineTo(f32.Pt(x1, y1-rad))
	p.CubeTo(f32.Pt(x1, y1-rad+rad*k), f32.Pt(x1-rad+rad*k, y1), f32.Pt(x1-rad, y1))
	p.LineTo(f32.Pt(x0+rad, y1))
	p.CubeTo(f32.Pt(x0+rad-rad*k, y1), f32.Pt(x0, y1-rad+rad*k), f32.Pt(x0, y1-rad))
	p.LineTo(f32.Pt(x0, y0+rad))
	p.CubeTo(f32.Pt(x0, y0+rad-rad*k), f32.Pt(x0+rad-rad*k, y0), f32.Pt(x0+rad, y0))
	p.Close()
	return p.End()
}

// box is a rounded rectangle with a fill and a border, the mocks' cards,
// fields and buttons.
type box struct {
	fill   color.NRGBA
	border color.NRGBA
	bw     float32 // border width in dp
	radius float32
	ring   color.NRGBA // an outer ring, the focus ring
	ringW  float32
}

func (b box) paint(gtx layout.Context, sz image.Point) {
	r := image.Rectangle{Max: sz}
	rad := dp(gtx, b.radius)
	if b.ring.A > 0 && b.ringW > 0 {
		rw := dp(gtx, b.ringW)
		fillRRect(gtx, r.Inset(-rw), rad+rw, b.ring)
	}
	fillRRect(gtx, r, rad, b.fill)
	strokeRRect(gtx, r, rad, fdp(gtx, b.bw), b.border)
}

// layout draws the box behind w, which gets the full constraints.
func (b box) layout(gtx layout.Context, w layout.Widget) layout.Dimensions {
	m := op.Record(gtx.Ops)
	dims := w(gtx)
	c := m.Stop()
	b.paint(gtx, dims.Size)
	c.Add(gtx.Ops)
	return dims
}

// polygon fills the polygon pts, given in a 100x100 box scaled to size px.
func polygon(gtx layout.Context, pts []f32.Point, scale float32, off f32.Point, c color.NRGBA) {
	var p clip.Path
	p.Begin(gtx.Ops)
	for i, pt := range pts {
		q := f32.Pt(pt.X*scale+off.X, pt.Y*scale+off.Y)
		if i == 0 {
			p.MoveTo(q)
		} else {
			p.LineTo(q)
		}
	}
	p.Close()
	paint.FillShape(gtx.Ops, c, clip.Outline{Path: p.End()}.Op())
}

// The six-triangle Isotope mark, from tools/gen-wallpaper.py by way of the
// mocks, in a 100x100 box.
type tri struct {
	pts []f32.Point
	col color.NRGBA
}

func markTris(lit bool) []tri {
	t := []tri{
		{[]f32.Point{{X: 54.7, Y: 41.9}, {X: 54.7, Y: 4.3}, {X: 87.3, Y: 23.1}}, colGlow},
		{[]f32.Point{{X: 51.4, Y: 50.0}, {X: 83.9, Y: 31.2}, {X: 83.9, Y: 68.8}}, colAccent},
		{[]f32.Point{{X: 50.7, Y: 51.2}, {X: 83.3, Y: 70.0}, {X: 50.7, Y: 88.8}}, colCheck},
		{[]f32.Point{{X: 49.3, Y: 51.2}, {X: 49.3, Y: 88.8}, {X: 16.7, Y: 70.0}}, colGlow},
		{[]f32.Point{{X: 48.6, Y: 50.0}, {X: 16.1, Y: 68.8}, {X: 16.1, Y: 31.2}}, colAccent},
		{[]f32.Point{{X: 49.3, Y: 48.8}, {X: 16.7, Y: 30.0}, {X: 49.3, Y: 11.2}}, colCheck},
	}
	if lit {
		// The installing page: the first triangle pushed out and lit, as
		// the splash does at the flash.
		t[0] = tri{[]f32.Point{{X: 58.7, Y: 39.6}, {X: 58.7, Y: 2.0}, {X: 91.3, Y: 20.8}}, colText}
	}
	return t
}

// mark draws the mark in a square of size dp. lit draws the installing
// page's variant, whose viewBox is -8..108 so the pushed triangle fits.
func mark(gtx layout.Context, size float32, lit bool, opacity float32) layout.Dimensions {
	px := fdp(gtx, size)
	scale, off := px/100, f32.Pt(0, 0)
	if lit {
		scale = px / 116
		off = f32.Pt(8*scale, 8*scale)
	}
	for _, t := range markTris(lit) {
		c := t.col
		if opacity < 1 {
			c = over(colDesk2, c, opacity)
		}
		polygon(gtx, t.pts, scale, off, c)
	}
	s := int(px + 0.5)
	return layout.Dimensions{Size: image.Pt(s, s)}
}

// glow paints a soft round light of colour c, centred in a square of size dp,
// fading to nothing at its edge: the mocks' drop-shadows and radial glows.
func glow(gtx layout.Context, size float32, c color.NRGBA, peak float32) {
	px := dp(gtx, size)
	img := glowImage(px, c, peak)
	defer clip.Rect(image.Rectangle{Max: image.Pt(px, px)}).Push(gtx.Ops).Pop()
	paint.NewImageOp(img).Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
}

var glowCache = map[[3]int]*image.NRGBA{}

func glowImage(px int, c color.NRGBA, peak float32) *image.NRGBA {
	k := [3]int{px, int(c.R)<<16 | int(c.G)<<8 | int(c.B), int(peak * 1000)}
	if img, ok := glowCache[k]; ok {
		return img
	}
	img := image.NewNRGBA(image.Rect(0, 0, px, px))
	r := float64(px) / 2
	for y := 0; y < px; y++ {
		for x := 0; x < px; x++ {
			d := math.Hypot(float64(x)+0.5-r, float64(y)+0.5-r) / r
			if d >= 1 {
				continue
			}
			a := float64(peak) * (1 - d) * (1 - d)
			img.SetNRGBA(x, y, color.NRGBA{R: c.R, G: c.G, B: c.B, A: uint8(a*255 + 0.5)})
		}
	}
	glowCache[k] = img
	return img
}

// checkIcon is the mocks' done check: a filled circle with a tick.
func checkIcon(gtx layout.Context, size float32) layout.Dimensions {
	px := fdp(gtx, size)
	s := int(px + 0.5)
	fillRRect(gtx, image.Rect(0, 0, s, s), s/2, colCheck)
	k := px / 10
	var p clip.Path
	p.Begin(gtx.Ops)
	p.MoveTo(f32.Pt(2.8*k, 5.2*k))
	p.LineTo(f32.Pt(4.3*k, 6.6*k))
	p.LineTo(f32.Pt(7.2*k, 3.6*k))
	paint.FillShape(gtx.Ops, colText, clip.Stroke{Path: p.End(), Width: 1.3 * k}.Op())
	return layout.Dimensions{Size: image.Pt(s, s)}
}

// ringIcon is a later step: an empty circle.
func ringIcon(gtx layout.Context, size float32) layout.Dimensions {
	s := dp(gtx, size)
	strokeRRect(gtx, image.Rect(0, 0, s, s), s/2, fdp(gtx, 1.5), colRing)
	return layout.Dimensions{Size: image.Pt(s, s)}
}

// spinner is the installing page's current stage: a track and an arc that
// turns with phase (0..1).
func spinner(gtx layout.Context, size float32, phase float32) layout.Dimensions {
	px := fdp(gtx, size)
	s := int(px + 0.5)
	k := px / 12
	c := f32.Pt(6*k, 6*k)
	r := 4.5 * k
	var t clip.Path
	t.Begin(gtx.Ops)
	t.MoveTo(f32.Pt(c.X+r, c.Y))
	t.ArcTo(c, c, 2*math.Pi)
	paint.FillShape(gtx.Ops, colField, clip.Stroke{Path: t.End(), Width: 2 * k}.Op())
	a0 := float64(phase)*2*math.Pi - math.Pi/2
	var p clip.Path
	p.Begin(gtx.Ops)
	p.MoveTo(f32.Pt(c.X+r*float32(math.Cos(a0)), c.Y+r*float32(math.Sin(a0))))
	p.ArcTo(c, c, math.Pi/2)
	paint.FillShape(gtx.Ops, colAccent, clip.Stroke{Path: p.End(), Width: 2 * k}.Op())
	return layout.Dimensions{Size: image.Pt(s, s)}
}

// warnIcon is the triangle with an exclamation mark the red warnings carry,
// so they do not rely on colour alone.
func warnIcon(gtx layout.Context, size float32, c color.NRGBA) layout.Dimensions {
	px := fdp(gtx, size)
	k := px / 16
	var p clip.Path
	p.Begin(gtx.Ops)
	p.MoveTo(f32.Pt(8*k, 1.5*k))
	p.LineTo(f32.Pt(15*k, 14*k))
	p.LineTo(f32.Pt(1*k, 14*k))
	p.Close()
	paint.FillShape(gtx.Ops, c, clip.Stroke{Path: p.End(), Width: 1.4 * k}.Op())
	var q clip.Path
	q.Begin(gtx.Ops)
	q.MoveTo(f32.Pt(8*k, 6*k))
	q.LineTo(f32.Pt(8*k, 10*k))
	q.MoveTo(f32.Pt(8*k, 11.5*k))
	q.LineTo(f32.Pt(8*k, 12.5*k))
	paint.FillShape(gtx.Ops, c, clip.Stroke{Path: q.End(), Width: 1.4 * k}.Op())
	s := int(px + 0.5)
	return layout.Dimensions{Size: image.Pt(s, s)}
}

// searchIcon is the magnifier in the layout search box.
func searchIcon(gtx layout.Context, size float32) layout.Dimensions {
	px := fdp(gtx, size)
	k := px / 16
	var p clip.Path
	p.Begin(gtx.Ops)
	c := f32.Pt(7*k, 7*k)
	p.MoveTo(f32.Pt(12*k, 7*k))
	p.ArcTo(c, c, 2*math.Pi)
	p.MoveTo(f32.Pt(11*k, 11*k))
	p.LineTo(f32.Pt(14*k, 14*k))
	paint.FillShape(gtx.Ops, colDim, clip.Stroke{Path: p.End(), Width: 1.5 * k}.Op())
	s := int(px + 0.5)
	return layout.Dimensions{Size: image.Pt(s, s)}
}

// keyIcon is the key beside "SSH keys from GitHub".
func keyIcon(gtx layout.Context, size float32) layout.Dimensions {
	px := fdp(gtx, size)
	k := px / 16
	var p clip.Path
	p.Begin(gtx.Ops)
	c := f32.Pt(5*k, 6*k)
	p.MoveTo(f32.Pt(8*k, 6*k))
	p.ArcTo(c, c, 2*math.Pi)
	p.MoveTo(f32.Pt(8*k, 6*k))
	p.LineTo(f32.Pt(15*k, 6*k))
	p.MoveTo(f32.Pt(13*k, 6*k))
	p.LineTo(f32.Pt(13*k, 9*k))
	p.MoveTo(f32.Pt(11*k, 6*k))
	p.LineTo(f32.Pt(11*k, 8.5*k))
	paint.FillShape(gtx.Ops, colGlow, clip.Stroke{Path: p.End(), Width: 1.2 * k}.Op())
	s := int(px + 0.5)
	return layout.Dimensions{Size: image.Pt(s, s)}
}

// diskIcon is the drive drawn on each disk card. Telling a USB stick from a
// drive needs the bus from sysfs, which comes with step 3 of the design note.
func diskIcon(gtx layout.Context, size float32, selected bool) layout.Dimensions {
	px := fdp(gtx, size)
	k := px / 48
	edge, led := colRing, colRing
	if selected {
		edge, led = colGlow, colAccent
	}
	rect := func(x, y, w, h float32) image.Rectangle {
		return image.Rect(int(x*k+0.5), int(y*k+0.5), int((x+w)*k+0.5), int((y+h)*k+0.5))
	}
	r := rect(6, 10, 36, 28)
	fillRRect(gtx, r, int(5*k), colField)
	strokeRRect(gtx, r, int(5*k), 2*k, edge)
	d := int(5 * k)
	cx, cy := int(34*k), int(31*k)
	fillRRect(gtx, image.Rect(cx-d/2, cy-d/2, cx+d/2+1, cy+d/2+1), d/2+1, led)
	fillRect(gtx, rect(12, 17, 18, 2), colRing)
	s := int(px + 0.5)
	return layout.Dimensions{Size: image.Pt(s, s)}
}

// desktop is the background behind the window: the radial glow of the
// mocks' radial-gradient(ellipse 90% 70% at 50% -15%) and, when the window
// is floating, its shadow (0 40px 120px rgba(0,0,0,.55)). It is drawn on the
// CPU once per size; Gio has no radial gradients or blurs.
type desktop struct {
	size, win image.Rectangle
	shadow    bool
	img       *image.RGBA
}

func (d *desktop) image(size image.Point, win image.Rectangle, shadow bool, pxPerDp float32) *image.RGBA {
	r := image.Rectangle{Max: size}
	if d.img != nil && d.size == r && d.win == win && d.shadow == shadow {
		return d.img
	}
	img := image.NewRGBA(r)
	w, h := float64(size.X), float64(size.Y)
	cx, cy, rx, ry := w*0.5, -0.15*h, 0.9*w, 0.7*h
	sigma := 60 * float64(pxPerDp)
	offY := 40 * float64(pxPerDp)
	erf := func(v float64) float64 { return math.Erf(v / (sigma * math.Sqrt2)) }
	// The shadow is separable: a rectangle blurred by a Gaussian is the
	// product of two 1D blurred steps.
	sx := make([]float64, size.X)
	sy := make([]float64, size.Y)
	if shadow {
		for x := range sx {
			fx := float64(x) + 0.5
			sx[x] = 0.5 * (erf(fx-float64(win.Min.X)) - erf(fx-float64(win.Max.X)))
		}
		for y := range sy {
			fy := float64(y) + 0.5 - offY
			sy[y] = 0.5 * (erf(fy-float64(win.Min.Y)) - erf(fy-float64(win.Max.Y)))
		}
	}
	c1, c2, c3 := colDesk1, colDesk2, colDesk3
	lerp := func(a, b uint8, t float64) float64 { return float64(a) + (float64(b)-float64(a))*t }
	for y := 0; y < size.Y; y++ {
		for x := 0; x < size.X; x++ {
			dx, dy := (float64(x)+0.5-cx)/rx, (float64(y)+0.5-cy)/ry
			t := math.Sqrt(dx*dx + dy*dy)
			var r, g, b float64
			switch {
			case t <= 0.55:
				u := t / 0.55
				r, g, b = lerp(c1.R, c2.R, u), lerp(c1.G, c2.G, u), lerp(c1.B, c2.B, u)
			case t < 1:
				u := (t - 0.55) / 0.45
				r, g, b = lerp(c2.R, c3.R, u), lerp(c2.G, c3.G, u), lerp(c2.B, c3.B, u)
			default:
				r, g, b = float64(c3.R), float64(c3.G), float64(c3.B)
			}
			if shadow {
				a := 0.55 * sx[x] * sy[y]
				r, g, b = r*(1-a), g*(1-a), b*(1-a)
			}
			i := img.PixOffset(x, y)
			img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = uint8(r+0.5), uint8(g+0.5), uint8(b+0.5), 0xff
		}
	}
	d.size, d.win, d.shadow, d.img = r, win, shadow, img
	return img
}

// tick is a check mark alone, in a size dp box.
func tick(gtx layout.Context, size float32, c color.NRGBA) {
	k := fdp(gtx, size) / 16
	var p clip.Path
	p.Begin(gtx.Ops)
	p.MoveTo(f32.Pt(3.5*k, 8.5*k))
	p.LineTo(f32.Pt(6.5*k, 11.5*k))
	p.LineTo(f32.Pt(12.5*k, 4.5*k))
	paint.FillShape(gtx.Ops, c, clip.Stroke{Path: p.End(), Width: 2 * k}.Op())
}
