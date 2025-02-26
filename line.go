package gvcode

import (
	"fmt"
	"image"
	"iter"

	"gioui.org/text"
	"golang.org/x/image/math/fixed"
)

// A line contains various metrics of a line of text.
type line struct {
	xOff    fixed.Int26_6
	yOff    int
	width   fixed.Int26_6
	ascent  fixed.Int26_6
	descent fixed.Int26_6
	glyphs  []*text.Glyph
	// runes is the number of runes represented by this line.
	runes int
	// runeOff tracks the rune offset of the first rune of the line in the document.
	runeOff int
}

func (li line) String() string {
	return fmt.Sprintf("[line] xOff: %d, yOff: %d, width: %d, runes: %d, runeOff: %d, glyphs: %d",
		li.xOff.Round(), li.yOff, li.width.Ceil(), li.runes, li.runeOff, len(li.glyphs))
}

func (li *line) append(glyphs ...text.Glyph) {
	for _, gl := range glyphs {
		li.yOff = int(gl.Y)
		if li.xOff > gl.X {
			li.xOff = gl.X
		}

		li.width += gl.Advance
		// glyph ascent and descent are derived from the line ascent and descent,
		// so it is safe to just set them as the line's ascent and descent.
		li.ascent = gl.Ascent
		li.descent = gl.Descent
		li.runes += int(gl.Runes)
		li.glyphs = append(li.glyphs, &gl)
	}

}

// recompute re-compute xOff of the line and each glyph contained in the line,
// and also update the runeOff to the right value.
func (li *line) recompute(alignOff fixed.Int26_6, runeOff int) {
	xOff := fixed.I(0)
	for idx, gl := range li.glyphs {
		gl.X = alignOff + fixed.Int26_6(xOff)
		if idx == len(li.glyphs)-1 {
			gl.Flags |= text.FlagLineBreak
		}

		xOff += gl.Advance
	}

	li.runeOff = runeOff
}

func (li *line) adjustYOff(yOff int) {
	li.yOff = yOff
	for _, gl := range li.glyphs {
		gl.Y = int32(yOff)
	}
}

func (li *line) bounds() image.Rectangle {
	return image.Rectangle{
		Min: image.Pt(li.xOff.Floor(), li.yOff-li.ascent.Ceil()),
		Max: image.Pt((li.xOff + li.width).Ceil(), li.yOff+li.descent.Ceil()),
	}
}

func (li *line) getGlyphs(offset, count int) []text.Glyph {
	if count <= 0 {
		return []text.Glyph{}
	}

	out := make([]text.Glyph, count)
	for idx, gl := range li.glyphs[offset : offset+count] {
		out[idx] = *gl
	}

	return out
}

// paragraph contains the pixel coordinates of the start and end position
// of the paragraph.
type paragraph struct {
	startX fixed.Int26_6
	startY int
	endX   fixed.Int26_6
	endY   int
	// runes is the number of runes represented by this paragraph.
	runes int
	// runeOff tracks the rune offset of the first rune of the paragraph in the document.
	runeOff int
}

// Add add a visual line to the paragraph, returning a boolean value indicating
// the end of a paragraph.
func (p *paragraph) Add(l *line) bool {
	lastGlyph := l.glyphs[len(l.glyphs)-1]

	if p.runes == 0 {
		start := l.glyphs[0]
		p.startX = start.X
		p.startY = int(start.Y)

		p.endX = lastGlyph.X
		p.endY = int(lastGlyph.Y)

		p.runeOff = l.runeOff
	} else {
		p.endX = lastGlyph.X
		p.endY = int(lastGlyph.Y)
	}

	p.runes += l.runes
	return lastGlyph.Flags&text.FlagParagraphBreak != 0
}

type glyphIter struct {
	shaper *text.Shaper
}

func (gi glyphIter) All() iter.Seq[text.Glyph] {
	return func(yield func(text.Glyph) bool) {
		for {
			g, ok := gi.shaper.NextGlyph()
			if !ok {
				return
			}

			if !yield(g) {
				return
			}
		}
	}
}
