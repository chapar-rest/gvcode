package painter

import (
	"image"
	"iter"

	"gioui.org/op"
	"gioui.org/text"
	lt "github.com/oligo/gvcode/internal/layout"
	"github.com/oligo/gvcode/textstyle/syntax"
	"golang.org/x/image/math/fixed"
)

type textStyle uint8

const (
	underline textStyle = 1 << iota
	squiggle
	strikethrough
	border
)

func (s textStyle) hasStyle(mask textStyle) bool {
	return s&mask > 0
}

// A glyphSpan is a group of adjacent glyphs sharing the same fg and bg.
type glyphSpan struct {
	// start and end is the glyph offset of a Line.
	start, end int
	fg         op.CallOp
	bg         op.CallOp
	style      syntax.TextStyle
	// offset is an visual offset relative to the start of the line.
	offset fixed.Int26_6
}

// bounds returns the bounding box relative to the dot of the first
// glyph of the span.
func (s *glyphSpan) bounds(line *lt.Line) image.Rectangle {
	rect := image.Rectangle{}

	if s.end-s.start <= 0 {
		return rect
	}

	for _, g := range line.Glyphs[s.start:s.end] {
		rect.Min.Y = min(rect.Min.Y, -g.Ascent.Round())
		rect.Max.Y = max(rect.Max.Y, g.Descent.Round())
		rect.Max.X += g.Advance.Round()
	}

	return rect
}

func (s *glyphSpan) width(line *lt.Line) fixed.Int26_6 {
	w := fixed.I(0)
	if s.end-s.start <= 0 {
		return w
	}

	for _, g := range line.Glyphs[s.start:s.end] {
		w += g.Advance
	}

	return w
}

func (s *glyphSpan) size() int {
	return s.end - s.start
}

// hasStyle checks whether the run should be paint with the specified style.
func (s *glyphSpan) hasStyle(mask syntax.TextStyle) bool {
	return s.style.HasStyle(mask)
}

// lineSplitter split line into runs with the same style.
type lineSplitter struct {
	current   glyphSpan
	runeOff   int
	advance   fixed.Int26_6
	nextGlyph func() (text.Glyph, bool)
	stopFunc  func()
	runs      []glyphSpan
}

func (rb *lineSplitter) setup(line *lt.Line) {
	lineIter := line.All()
	rb.nextGlyph, rb.stopFunc = iter.Pull(lineIter)
	rb.current = glyphSpan{}
	rb.advance = fixed.I(0)
}

func (rb *lineSplitter) commitLast() {
	if rb.current.size() > 0 {
		rb.runs = append(rb.runs, rb.current)
		rb.current = glyphSpan{
			start:  rb.current.end,
			end:    rb.current.end,
			offset: rb.advance,
		}
	}
}

func (rb *lineSplitter) Split(line *lt.Line, textTokens *syntax.TextTokens) {
	rb.runs = rb.runs[:0]
	rb.runeOff = line.RuneOff

	tokens := textTokens.QueryRange(line.RuneOff, line.RuneOff+line.Runes)
	if len(tokens) == 0 {
		run := glyphSpan{
			start:  0,
			end:    len(line.Glyphs),
			offset: 0,
		}

		rb.runs = append(rb.runs, run)
		return
	}

	rb.setup(line)
	defer rb.stopFunc()

	for _, token := range tokens {
		// check if there is any glyphs not covered by the token and put them in
		// one run.
		rb.readUntil(token.Start)
		if rb.current.size() > 0 {
			// no style
			rb.commitLast()
		}

		// next read the entire token range to the current run.
		rb.readUntil(token.End)
		if rb.current.size() > 0 {
			bgId := token.Style.Background()
			fgId := token.Style.Foreground()
			rb.current.fg = textTokens.GetColor(fgId)
			rb.current.bg = textTokens.GetColor(bgId)
			rb.current.style = token.Style.FontStyle()
			rb.commitLast()
		}
	}

	if rb.current.size() > 0 {
		rb.commitLast()
	}

}

func (rb *lineSplitter) readUntil(runeOff int) {
	for rb.runeOff < runeOff {
		g, ok := rb.nextGlyph()
		if !ok {
			break
		}
		rb.advance += g.Advance
		rb.current.end += 1
		rb.runeOff += int(g.Runes)
	}
}

func (rb *lineSplitter) Runs() iter.Seq[glyphSpan] {
	return func(yield func(glyphSpan) bool) {
		for _, run := range rb.runs {
			if !yield(run) {
				return
			}
		}
	}
}

func (rb *lineSplitter) Size() int {
	return len(rb.runs)
}
