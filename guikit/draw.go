package guikit

import (
	"image"
	"image/color"
	"math"
	"time"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
)

// Dp converts the mocks' CSS pixels, which are logical pixels, to device
// pixels.
func Dp(gtx layout.Context, v float32) int { return gtx.Dp(unit.Dp(v)) }

// Fdp is Dp without the rounding to whole pixels, to a 64th of one.
func Fdp(gtx layout.Context, v float32) float32 { return float32(gtx.Dp(unit.Dp(v*64))) / 64 }

// Pt is image.Pt in dp.
func Pt(gtx layout.Context, x, y float32) image.Point { return image.Pt(Dp(gtx, x), Dp(gtx, y)) }

// FillRect fills r with c.
func FillRect(gtx layout.Context, r image.Rectangle, c color.NRGBA) {
	paint.FillShape(gtx.Ops, c, clip.Rect(r).Op())
}

// FillRRect fills r, with corners of radius rad, with c.
func FillRRect(gtx layout.Context, r image.Rectangle, rad int, c color.NRGBA) {
	paint.FillShape(gtx.Ops, c, clip.UniformRRect(r, rad).Op(gtx.Ops))
}

// StrokeRRect draws a border of width w inside r.
func StrokeRRect(gtx layout.Context, r image.Rectangle, rad int, w float32, c color.NRGBA) {
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

// Box is a rounded rectangle with a fill and a border: the mocks' cards,
// fields and buttons. Sizes are in dp.
type Box struct {
	Fill        color.NRGBA
	Border      color.NRGBA
	BorderWidth float32
	Radius      float32
	Ring        color.NRGBA // an outer ring, the focus ring
	RingWidth   float32
}

// Paint draws the box at size sz.
func (b Box) Paint(gtx layout.Context, sz image.Point) {
	r := image.Rectangle{Max: sz}
	rad := Dp(gtx, b.Radius)
	if b.Ring.A > 0 && b.RingWidth > 0 {
		rw := Dp(gtx, b.RingWidth)
		FillRRect(gtx, r.Inset(-rw), rad+rw, b.Ring)
	}
	FillRRect(gtx, r, rad, b.Fill)
	StrokeRRect(gtx, r, rad, Fdp(gtx, b.BorderWidth), b.Border)
}

// Layout draws the box behind w, which gets the full constraints.
func (b Box) Layout(gtx layout.Context, w layout.Widget) layout.Dimensions {
	m := op.Record(gtx.Ops)
	dims := w(gtx)
	c := m.Stop()
	b.Paint(gtx, dims.Size)
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

func markTris(lit bool, push float32) []tri {
	t := []tri{
		{[]f32.Point{{X: 54.7, Y: 41.9}, {X: 54.7, Y: 4.3}, {X: 87.3, Y: 23.1}}, ColGlow},
		{[]f32.Point{{X: 51.4, Y: 50.0}, {X: 83.9, Y: 31.2}, {X: 83.9, Y: 68.8}}, ColAccent},
		{[]f32.Point{{X: 50.7, Y: 51.2}, {X: 83.3, Y: 70.0}, {X: 50.7, Y: 88.8}}, ColCheck},
		{[]f32.Point{{X: 49.3, Y: 51.2}, {X: 49.3, Y: 88.8}, {X: 16.7, Y: 70.0}}, ColGlow},
		{[]f32.Point{{X: 48.6, Y: 50.0}, {X: 16.1, Y: 68.8}, {X: 16.1, Y: 31.2}}, ColAccent},
		{[]f32.Point{{X: 49.3, Y: 48.8}, {X: 16.7, Y: 30.0}, {X: 49.3, Y: 11.2}}, ColCheck},
	}
	if lit {
		// The first triangle pushed out and lit, as the splash does at
		// the flash.
		t[0].col = ColText
	}
	// The push moves the first triangle out along the line from the
	// centre, (4, -2.3) per unit: the mocks' installing mark is one unit.
	pts := make([]f32.Point, 3)
	for i, p := range t[0].pts {
		pts[i] = f32.Pt(p.X+4*push, p.Y-2.3*push)
	}
	t[0].pts = pts
	return t
}

// Breath is the splash's breathing of the pushed triangle: a slow sine,
// 2.4 s a cycle, at time t, from -1 to 1.
func Breath(t time.Time) float32 {
	s := float64(t.UnixMilli()%2400) / 2400
	return float32(math.Sin(s * 2 * math.Pi))
}

// Mark draws the Isotope mark in a square of size dp, at opacity over the
// desktop colour. lit draws the busy variant (the installer's installing
// page): the first triangle in white, in a viewBox of -8..108 so it fits
// when pushed; push is how far out the first triangle is, 0 for the plain
// mark.
func Mark(gtx layout.Context, size float32, lit bool, opacity, push float32) layout.Dimensions {
	px := Fdp(gtx, size)
	scale, off := px/100, f32.Pt(0, 0)
	if lit {
		scale = px / 116
		off = f32.Pt(8*scale, 8*scale)
	}
	for _, t := range markTris(lit, push) {
		c := t.col
		if opacity < 1 {
			c = Over(ColDesk2, c, opacity)
		}
		polygon(gtx, t.pts, scale, off, c)
	}
	s := int(px + 0.5)
	return layout.Dimensions{Size: image.Pt(s, s)}
}

// Glow paints a soft round light of colour c, centred in a square of size
// dp, fading to nothing at its edge: the mocks' drop-shadows and radial
// glows. peak is the opacity at the centre.
func Glow(gtx layout.Context, size float32, c color.NRGBA, peak float32) {
	px := Dp(gtx, size)
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

// CheckIcon is the mocks' done check: a filled circle with a tick.
func CheckIcon(gtx layout.Context, size float32) layout.Dimensions {
	px := Fdp(gtx, size)
	s := int(px + 0.5)
	FillRRect(gtx, image.Rect(0, 0, s, s), s/2, ColCheck)
	k := px / 10
	var p clip.Path
	p.Begin(gtx.Ops)
	p.MoveTo(f32.Pt(2.8*k, 5.2*k))
	p.LineTo(f32.Pt(4.3*k, 6.6*k))
	p.LineTo(f32.Pt(7.2*k, 3.6*k))
	paint.FillShape(gtx.Ops, ColText, clip.Stroke{Path: p.End(), Width: 1.3 * k}.Op())
	return layout.Dimensions{Size: image.Pt(s, s)}
}

// RingIcon is a step still to come: an empty circle.
func RingIcon(gtx layout.Context, size float32) layout.Dimensions {
	s := Dp(gtx, size)
	StrokeRRect(gtx, image.Rect(0, 0, s, s), s/2, Fdp(gtx, 1.5), ColRing)
	return layout.Dimensions{Size: image.Pt(s, s)}
}

// Spinner is work in progress: a track and an arc that turns with phase
// (0..1).
func Spinner(gtx layout.Context, size float32, phase float32) layout.Dimensions {
	px := Fdp(gtx, size)
	s := int(px + 0.5)
	k := px / 12
	c := f32.Pt(6*k, 6*k)
	r := 4.5 * k
	var t clip.Path
	t.Begin(gtx.Ops)
	t.MoveTo(f32.Pt(c.X+r, c.Y))
	t.ArcTo(c, c, 2*math.Pi)
	paint.FillShape(gtx.Ops, ColField, clip.Stroke{Path: t.End(), Width: 2 * k}.Op())
	a0 := float64(phase)*2*math.Pi - math.Pi/2
	var p clip.Path
	p.Begin(gtx.Ops)
	p.MoveTo(f32.Pt(c.X+r*float32(math.Cos(a0)), c.Y+r*float32(math.Sin(a0))))
	p.ArcTo(c, c, math.Pi/2)
	paint.FillShape(gtx.Ops, ColAccent, clip.Stroke{Path: p.End(), Width: 2 * k}.Op())
	return layout.Dimensions{Size: image.Pt(s, s)}
}

// WarnIcon is the triangle with an exclamation mark that warnings carry, so
// they do not rely on colour alone.
func WarnIcon(gtx layout.Context, size float32, c color.NRGBA) layout.Dimensions {
	px := Fdp(gtx, size)
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

// Tick is a check mark alone, in a size dp box.
func Tick(gtx layout.Context, size float32, c color.NRGBA) {
	k := Fdp(gtx, size) / 16
	var p clip.Path
	p.Begin(gtx.Ops)
	p.MoveTo(f32.Pt(3.5*k, 8.5*k))
	p.LineTo(f32.Pt(6.5*k, 11.5*k))
	p.LineTo(f32.Pt(12.5*k, 4.5*k))
	paint.FillShape(gtx.Ops, c, clip.Stroke{Path: p.End(), Width: 2 * k}.Op())
}

// ProgressBar is the 8 dp bar, filled to f (0..1): green to glow, as the
// mocks' gradient.
func ProgressBar(gtx layout.Context, f float32) layout.Dimensions {
	w, h := gtx.Constraints.Max.X, Dp(gtx, 8)
	r := Dp(gtx, 4)
	FillRRect(gtx, image.Rect(0, 0, w, h), r, ColField)
	fw := int(float32(w) * f)
	if fw > 0 {
		cl := clip.UniformRRect(image.Rect(0, 0, fw, h), r).Push(gtx.Ops)
		paint.LinearGradientOp{
			Stop1: f32.Pt(0, 0), Color1: ColCheck,
			Stop2: f32.Pt(float32(fw)*0.7, 0), Color2: ColAccent,
		}.Add(gtx.Ops)
		paint.PaintOp{}.Add(gtx.Ops)
		cl.Pop()
	}
	return layout.Dimensions{Size: image.Pt(w, h)}
}
