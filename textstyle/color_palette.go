package textstyle

import (
	"image/color"
	"slices"

	"gioui.org/op"
	"gioui.org/op/paint"
)

// Color wraps a color.NRGBA color which is widely used by Gio.
// It provides method to convert the non-alpha-premultiplied color
// to a color OP used by Gio ops.
type Color struct {
	val color.NRGBA
	op  op.CallOp
}

func (c *Color) NRGBA() color.NRGBA {
	return c.val
}

func (c *Color) makeOp() {
	if c.op != (op.CallOp{}) {
		return
	}
	ops := new(op.Ops)
	m := op.Record(ops)
	paint.ColorOp{Color: c.val}.Add(ops)
	c.op = m.Stop()
}

func (c *Color) Op() op.CallOp {
	if c.val == (color.NRGBA{}) {
		return op.CallOp{}
	}

	c.makeOp()
	return c.op
}

// ColorPalette manages used color of TextPainter. Color is added and referenced by its
// ID(index) in the palette.
type ColorPalette struct {
	colors []*Color
}

// GetColor retrieves a Color by its ID. ID can be acquired when adding the color to
// the palette.
func (p *ColorPalette) GetColor(id int) Color {
	if id < 0 || id >= len(p.colors) {
		return Color{}
	}

	c := p.colors[id]
	c.makeOp()
	return *c
}

// AddColor adds a color to the palette and return its id(index).
func (p *ColorPalette) AddColor(cl color.NRGBA) int {
	if idx := slices.IndexFunc(p.colors, func(c *Color) bool { return c.val == cl }); idx >= 0 {
		return idx
	}

	p.colors = append(p.colors, &Color{val: cl})
	return len(p.colors) - 1
}

// Clear clear all added colors.
func (p *ColorPalette) Clear() {
	p.colors = p.colors[:0]
}
