package editor

import (
	"image"
	"strconv"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
)

func paintLineNumber(gtx layout.Context, shaper *text.Shaper, params text.Parameters, viewport image.Rectangle, lines []int32, textMaterial op.CallOp) layout.Dimensions {
	// inherit all other settings from the main text layout.
	params.Alignment = text.End
	params.MinWidth = gtx.Constraints.Min.X
	params.MaxWidth = gtx.Constraints.Max.X

	var dims layout.Dimensions
	glyphs := make([]text.Glyph, 5)

	quit := false
lineLoop:
	for i, yOffset := range lines {
		if quit {
			break
		}

		if i == len(lines)-1 {
			// todo: adjust line height of the last empty line.
		}

		shaper.LayoutString(params, strconv.Itoa(i+1))
		glyphs = glyphs[:0]

		var bounds image.Rectangle
		visible := false
		for {
			g, ok := shaper.NextGlyph()
			if !ok {
				break
			}

			if int(yOffset)+g.Descent.Ceil() < viewport.Min.Y {
				break
			} else if int(yOffset)-g.Ascent.Ceil() > viewport.Max.Y {
				quit = true
				goto lineLoop
			}

			bounds.Min.X = min(bounds.Min.X, g.X.Floor())
			bounds.Min.Y = min(bounds.Min.Y, int(g.Y)-g.Ascent.Floor())
			bounds.Max.X = max(bounds.Max.X, (g.X + g.Advance).Ceil())
			bounds.Max.Y = max(bounds.Max.Y, int(g.Y)+g.Descent.Ceil())

			glyphs = append(glyphs, g)
			visible = true
		}

		if !visible {
			continue
		}

		dims.Size = image.Point{X: max(bounds.Dx(), dims.Size.X), Y: dims.Size.Y + bounds.Dy()}

		trans := op.Affine(f32.Affine2D{}.Offset(f32.Point{Y: float32(yOffset)}.Sub(layout.FPt(viewport.Min)))).Push(gtx.Ops)

		// draw glyph
		path := shaper.Shape(glyphs)
		outline := clip.Outline{Path: path}.Op().Push(gtx.Ops)
		textMaterial.Add(gtx.Ops)
		paint.PaintOp{}.Add(gtx.Ops)
		outline.Pop()
		trans.Pop()
	}

	dims.Size = gtx.Constraints.Constrain(dims.Size)
	return dims
}
