package painter

import (
	"image"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	lt "github.com/oligo/gvcode/internal/layout"
	"github.com/oligo/gvcode/textstyle/decoration"
	"github.com/oligo/gvcode/textstyle/syntax"
	"golang.org/x/image/math/fixed"
)

// TextPainter computes the bounding box of and paints text.
type TextPainter struct {
	// viewport is the rectangle of document coordinates that the painter is
	// trying to fill with text.
	viewport  image.Rectangle
	scrollOff image.Point

	// textTokens are tokens produced by language lexers for syntax highlighting.
	textTokens *syntax.TextTokens
	// Extra styles used to decorate the text.
	decorations *decoration.DecorationTree

	// padding is the space needed outside of the bounds of the text to ensure no
	// part of a glyph is clipped.
	padding image.Rectangle

	lineSplitter lineSplitter
}

func (tp *TextPainter) UpdateViewport(viewport image.Rectangle, scrollOff image.Point) {
	tp.viewport = viewport
	tp.scrollOff = scrollOff
}

func (tp *TextPainter) UpdateSyntaxTokens(textTokens *syntax.TextTokens) {
	tp.textTokens = textTokens
}

func (tp *TextPainter) SetDecorations() {
}

// processGlyph checks whether the glyph is visible within the configured
// viewport and (if so) updates the text dimensions to include the glyph.
func (tp *TextPainter) processGlyph(g text.Glyph) (visible bool) {
	// Compute the maximum extent to which glyphs overhang on the horizontal
	// axis.
	if d := g.Bounds.Min.X.Floor(); d < tp.padding.Min.X {
		// If the distance between the dot and the left edge of this glyph is
		// less than the current padding, increase the left padding.
		tp.padding.Min.X = d
	}
	if d := (g.Bounds.Max.X - g.Advance).Ceil(); d > tp.padding.Max.X {
		// If the distance between the dot and the right edge of this glyph
		// minus the logical advance of this glyph is greater than the current
		// padding, increase the right padding.
		tp.padding.Max.X = d
	}
	if d := (g.Bounds.Min.Y + g.Ascent).Floor(); d < tp.padding.Min.Y {
		// If the distance between the dot and the top of this glyph is greater
		// than the ascent of the glyph, increase the top padding.
		tp.padding.Min.Y = d
	}
	if d := (g.Bounds.Max.Y - g.Descent).Ceil(); d > tp.padding.Max.Y {
		// If the distance between the dot and the bottom of this glyph is greater
		// than the descent of the glyph, increase the bottom padding.
		tp.padding.Max.Y = d
	}
	logicalBounds := image.Rectangle{
		Min: image.Pt(g.X.Floor(), int(g.Y)-g.Ascent.Ceil()),
		Max: image.Pt((g.X + g.Advance).Ceil(), int(g.Y)+g.Descent.Ceil()),
	}

	above := logicalBounds.Max.Y < tp.viewport.Min.Y
	below := logicalBounds.Min.Y > tp.viewport.Max.Y
	left := logicalBounds.Max.X < tp.viewport.Min.X
	right := logicalBounds.Min.X > tp.viewport.Max.X

	return !above && !below && !left && !right
}

func (tp *TextPainter) PaintText(gtx layout.Context, shaper *text.Shaper, lines []*lt.Line, defaultMaterial op.CallOp) {
	m := op.Record(gtx.Ops)

	viewport := tp.viewport

	for _, line := range lines {
		if line.Descent.Ceil()+line.YOff < tp.viewport.Min.Y {
			continue
		}
		if line.YOff-line.Ascent.Floor() > tp.viewport.Max.Y {
			break
		}

		tp.paintLine(gtx, shaper, line, defaultMaterial)
	}

	call := m.Stop()
	viewport.Min = viewport.Min.Add(tp.padding.Min)
	viewport.Max = viewport.Max.Add(tp.padding.Max)
	// clip to make it fit the viewport.
	defer clip.Rect(viewport.Sub(tp.scrollOff)).Push(gtx.Ops).Pop()
	call.Add(gtx.Ops)
}

func (tp *TextPainter) paintLine(gtx layout.Context, shaper *text.Shaper, line *lt.Line, defaultMaterial op.CallOp) {
	if len(line.Glyphs) <= 0 {
		return
	}

	// Let drawing begin at the offset of the entire line.
	lineOff := f32.Point{X: fixedToFloat(line.XOff), Y: float32(line.YOff)}.Sub(layout.FPt(tp.viewport.Min))
	t := op.Affine(f32.Affine2D{}.Offset(lineOff)).Push(gtx.Ops)

	// split the line into runs.
	tp.lineSplitter.Split(line, tp.textTokens)
	// Iterate through the runs to paint the text.
	for run := range tp.lineSplitter.Runs() {
		// paint at the run offset.
		spanOffset := op.Affine(f32.Affine2D{}.Offset(f32.Point{X: float32(run.offset.Round())})).Push(gtx.Ops)

		glyphs := line.GetGlyphs(run.start, run.end)
		// draw background
		if run.bg != (op.CallOp{}) {
			rect := run.bounds(line)
			bgClip := clip.Rect(rect).Push(gtx.Ops)
			run.bg.Add(gtx.Ops)
			paint.PaintOp{}.Add(gtx.Ops)
			bgClip.Pop()
		}

		if run.style.HasStyle(syntax.Underline) {
			tp.drawUnderline(gtx, line, run, defaultMaterial)
		}
		if run.style.HasStyle(syntax.Strikethrough) {
			tp.drawStrikethrough(gtx, line, run, defaultMaterial)
		}
		if run.style.HasStyle(syntax.Border) {
			tp.drawBorder(gtx, line, run, defaultMaterial)
		}

		// draw glyph
		path := shaper.Shape(glyphs)
		outline := clip.Outline{Path: path}.Op().Push(gtx.Ops)
		if run.fg == (op.CallOp{}) {
			run.fg = defaultMaterial
		}
		run.fg.Add(gtx.Ops)
		paint.PaintOp{}.Add(gtx.Ops)
		outline.Pop()
		if call := shaper.Bitmaps(glyphs); call != (op.CallOp{}) {
			call.Add(gtx.Ops)
		}
		spanOffset.Pop()
	}

	t.Pop()
}

func (tp *TextPainter) drawDecorations(gtx layout.Context, shaper *text.Shaper, line *lt.Line, tokens []syntax.Token) {

}

func (tp *TextPainter) drawStroke(gtx layout.Context, path clip.PathSpec, material op.CallOp) {
	shape := clip.Stroke{
		Path:  path,
		Width: 1,
	}.Op()

	defer shape.Push(gtx.Ops).Pop()
	material.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
}

func (tp *TextPainter) drawUnderline(gtx layout.Context, line *lt.Line, run glyphSpan, material op.CallOp) {
	descent := line.Descent
	path := clip.Path{}
	path.Begin(gtx.Ops)
	path.Move(f32.Pt(fixedToFloat(run.offset), fixedToFloat(descent)))

	width := fixedToFloat(run.width(line))
	path.Line(f32.Point{X: width})
	path.Close()

	tp.drawStroke(gtx, path.End(), material)
}

func (tp *TextPainter) drawStrikethrough(gtx layout.Context, line *lt.Line, run glyphSpan, material op.CallOp) {
	path := clip.Path{}
	path.Begin(gtx.Ops)
	path.Move(f32.Point{X: fixedToFloat(run.offset)})

	width := fixedToFloat(run.width(line))
	path.Line(f32.Point{X: width})
	path.Close()

	tp.drawStroke(gtx, path.End(), material)
}

func (tp *TextPainter) drawBorder(gtx layout.Context, line *lt.Line, run glyphSpan, material op.CallOp) {
	rect := clip.Rect(run.bounds(line))
	tp.drawStroke(gtx, rect.Path(), material)
}

func fixedToFloat(i fixed.Int26_6) float32 {
	return float32(i) / 64.0
}
