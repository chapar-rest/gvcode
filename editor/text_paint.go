package editor

import (
	"bufio"
	"image"
	"io"
	"math"
	"sort"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
)

func (e *textView) layoutText(lt *text.Shaper) {
	e.rr.Seek(0, io.SeekStart)
	var r io.Reader = e.rr

	e.index.reset()
	it := textIterator{viewport: image.Rectangle{Max: image.Point{X: math.MaxInt, Y: math.MaxInt}}}
	if lt != nil {
		lt.Layout(e.params, r)
		for {
			g, ok := lt.NextGlyph()
			if !it.processGlyph(g, ok) {
				break
			}
			e.index.Glyph(g)
		}
	} else {
		// Make a fake glyph for every rune in the reader.
		b := bufio.NewReader(r)
		for _, _, err := b.ReadRune(); err != io.EOF; _, _, err = b.ReadRune() {
			g := text.Glyph{Runes: 1, Flags: text.FlagClusterBreak}
			_ = it.processGlyph(g, true)
			e.index.Glyph(g)
		}
	}
	e.paragraphReader.SetSource(e.rr)
	e.graphemes = e.graphemes[:0]
	for g := e.paragraphReader.Graphemes(); len(g) > 0; g = e.paragraphReader.Graphemes() {
		if len(e.graphemes) > 0 && g[0] == e.graphemes[len(e.graphemes)-1] {
			g = g[1:]
		}
		e.graphemes = append(e.graphemes, g...)
	}
	dims := layout.Dimensions{Size: it.bounds.Size()}
	dims.Baseline = dims.Size.Y - it.baseline
	e.dims = dims
}

// PaintText clips and paints the visible text glyph outlines using the provided
// material to fill the glyphs.
func (e *textView) PaintText(gtx layout.Context, material op.CallOp, textStyles []*TextStyle) {
	m := op.Record(gtx.Ops)
	viewport := image.Rectangle{
		Min: e.scrollOff,
		Max: e.viewSize.Add(e.scrollOff),
	}

	it := textIterator{
		viewport: viewport,
	}

	startGlyph := 0
	for _, line := range e.index.screenLines {
		if line.descent.Ceil()+line.yOff >= viewport.Min.Y {
			break
		}
		startGlyph += line.glyphs
	}
	var glyphs [32]glyphStyle
	line := glyphs[:0]
	start := startGlyph
	for _, g := range e.index.glyphs[startGlyph:] {
		var ok bool

		if line, ok = it.paintGlyph(gtx, e.shaper, e.styleForGlyph(g, material, textStyles), line); !ok {
			break
		}
		start++
	}

	call := m.Stop()
	viewport.Min = viewport.Min.Add(it.padding.Min)
	viewport.Max = viewport.Max.Add(it.padding.Max)
	defer clip.Rect(viewport.Sub(e.scrollOff)).Push(gtx.Ops).Pop()
	call.Add(gtx.Ops)
}

// PaintSelection clips and paints the visible text selection rectangles using
// the provided material to fill the rectangles.
func (e *textView) PaintSelection(gtx layout.Context, material op.CallOp) {
	localViewport := image.Rectangle{Max: e.viewSize}
	docViewport := image.Rectangle{Max: e.viewSize}.Add(e.scrollOff)
	defer clip.Rect(localViewport).Push(gtx.Ops).Pop()
	e.regions = e.index.locate(docViewport, e.caret.start, e.caret.end, e.regions)

	expandEmptyRegion := len(e.regions) > 1
	for _, region := range e.regions {
		bounds := e.adjustPadding(region.Bounds)
		if expandEmptyRegion && bounds.Dx() <= 0 {
			bounds.Max.X += gtx.Dp(unit.Dp(2))
		}
		area := clip.Rect(bounds).Push(gtx.Ops)
		material.Add(gtx.Ops)
		paint.PaintOp{}.Add(gtx.Ops)
		area.Pop()
	}
}

// paintRegions clips and paints the visible text rectangles using
// the provided material to fill the rectangles. Regions passed in should be constrained
// in the viewport.
func (e *textView) PaintRegions(gtx layout.Context, regions []Region, material op.CallOp) {
	localViewport := image.Rectangle{Max: e.viewSize}
	defer clip.Rect(localViewport).Push(gtx.Ops).Pop()
	for _, region := range regions {
		area := clip.Rect(e.adjustPadding(region.Bounds)).Push(gtx.Ops)
		material.Add(gtx.Ops)
		paint.PaintOp{}.Add(gtx.Ops)
		area.Pop()
	}
}

// caretCurrentLine returns the current logical line that the carent is in.
// Only the start position is checked.
func (e *textView) caretCurrentLine() (start combinedPos, end combinedPos) {
	caretStart := e.closestToRune(e.caret.start)
	if len(e.index.lineRanges) <= 0 {
		return caretStart, caretStart
	}

	lineIdx := sort.Search(len(e.index.lineRanges), func(i int) bool {
		rng := e.index.lineRanges[i]
		return rng.startY >= caretStart.y
	})

	// No exsiting lines found.
	if lineIdx == len(e.index.lineRanges) {
		return caretStart, caretStart
	}

	line := e.index.lineRanges[lineIdx]
	start = e.closestToXY(line.startX, line.startY)
	end = e.closestToXY(line.endX, line.endY)

	return
}

// paintLineHighlight clips and paints the visible line that the caret is in when there is no
// text selected.
func (e *textView) paintLineHighlight(gtx layout.Context, material op.CallOp) {
	if e.caret.start != e.caret.end {
		return
	}

	start, end := e.caretCurrentLine()
	if start == (combinedPos{}) || end == (combinedPos{}) {
		return
	}

	bounds := image.Rectangle{Min: image.Point{X: 0, Y: start.y - start.ascent.Ceil()},
		Max: image.Point{X: gtx.Constraints.Max.X, Y: end.y + end.descent.Ceil()}}.Sub(e.scrollOff)

	area := clip.Rect(e.adjustPadding(bounds)).Push(gtx.Ops)
	material.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	area.Pop()
}

func (e *textView) PaintLineNumber(gtx layout.Context, lt *text.Shaper, material op.CallOp) layout.Dimensions {
	m := op.Record(gtx.Ops)
	viewport := image.Rectangle{
		Min: e.scrollOff,
		Max: e.viewSize.Add(e.scrollOff),
	}

	dims := paintLineNumber(gtx, lt, e.params, viewport, e.index.lineRanges, material)
	call := m.Stop()

	rect := viewport.Sub(e.scrollOff)
	rect.Max.X = dims.Size.X
	defer clip.Rect(rect).Push(gtx.Ops).Pop()
	call.Add(gtx.Ops)

	return dims
}

// caretWidth returns the width occupied by the caret for the current gtx.
func (e *textView) caretWidth(gtx layout.Context) int {
	return gtx.Dp(1)
}

// PaintCaret clips and paints the caret rectangle, adding material immediately
// before painting to set the appropriate paint material.
func (e *textView) PaintCaret(gtx layout.Context, material op.CallOp) {
	carWidth2 := e.caretWidth(gtx)
	caretPos, carAsc, carDesc := e.CaretInfo()

	carRect := image.Rectangle{
		Min: caretPos.Sub(image.Pt(carWidth2, carAsc)),
		Max: caretPos.Add(image.Pt(carWidth2, carDesc)),
	}
	cl := image.Rectangle{Max: e.viewSize}
	carRect = cl.Intersect(carRect)
	if !carRect.Empty() {
		defer clip.Rect(e.adjustPadding(carRect)).Push(gtx.Ops).Pop()
		material.Add(gtx.Ops)
		paint.PaintOp{}.Add(gtx.Ops)
	}
}

func (e *textView) CaretInfo() (pos image.Point, ascent, descent int) {
	caretStart := e.closestToRune(e.caret.start)

	ascent = caretStart.ascent.Ceil()
	descent = caretStart.descent.Ceil()

	pos = image.Point{
		X: caretStart.x.Round(),
		Y: caretStart.y,
	}
	pos = pos.Sub(e.scrollOff)
	return
}

// Calculate line height. Maybe there's a better way?
func (e *textView) calcLineHeight() float32 {
	lineHeight := e.params.LineHeight
	// align with how text.Shaper handles default value of e.params.LineHeight.
	if lineHeight == 0 {
		lineHeight = e.params.PxPerEm
	}
	lineHeightScale := e.params.LineHeightScale
	// align with how text.Shaper handles default value of e.params.LineHeightScale.
	if lineHeightScale == 0 {
		lineHeightScale = 1.2
	}

	lh := float32(lineHeight.Round()) * lineHeightScale
	// log.Println("line height calculated: ", lineHeight.Ceil(), lh)
	return lh
}

// adjustPadding adjusts the vertical padding of a bounding box around the texts.
// This improves the visual effects of selected texts, or any other texts to be highlighted.
func (e *textView) adjustPadding(bounds image.Rectangle) image.Rectangle {
	if e.lineHeight <= 0 {
		e.lineHeight = e.calcLineHeight()
	}

	if e.lineHeight <= float32(bounds.Dy()) {
		return bounds
	}

	leading := e.lineHeight - float32(bounds.Dy())
	adjust := int(math.Round(float64(leading / 2.0)))
	lowerAdjust := int(math.Round(float64(leading - float32(adjust))))
	if adjust+lowerAdjust < int(leading) {
		lowerAdjust++
	}

	bounds.Min.Y -= adjust
	bounds.Max.Y += lowerAdjust
	return bounds
}

func (e *textView) styleForGlyph(g text.Glyph, detaultMaterial op.CallOp, styles []*TextStyle) glyphStyle {
	gs := glyphStyle{g: g}

	pos := e.index.closestToXY(g.X, int(g.Y))
	idx := sort.Search(len(styles), func(i int) bool {
		s := styles[i]
		return s.Start > pos.runes
	})

	if idx >= len(styles) {
		gs.fg = detaultMaterial
		return gs
	}

	if idx > 0 {
		idx--
	}

	style := styles[idx]
	gs.fg = style.Color
	gs.bg = style.Background

	if style.Color == (op.CallOp{}) {
		gs.fg = detaultMaterial
	}

	return gs
}
