// This file includes code from the Gio project, licensed under the MIT License.
// See the LICENSE file in the project root for more information.

package editor

import (
	"bufio"
	"fmt"
	"image"
	"io"

	"gioui.org/text"
	"github.com/go-text/typesetting/segmenter"
	"golang.org/x/image/math/fixed"
)

// screen line info
type lineInfo struct {
	xOff            fixed.Int26_6
	yOff            int
	width           fixed.Int26_6
	ascent, descent fixed.Int26_6
	glyphs          int
}

// lineRange contains the pixel coordinates of the start and end position
// of the logical line.
type lineRange struct {
	startX fixed.Int26_6
	startY int
	endX   fixed.Int26_6
	endY   int
}

// screenPos represents a character position in text line and column numbers,
// not pixels.
type screenPos struct {
	// col is the column, measured in runes.
	col  int
	line int
}

// combinedPos is a point in the editor.
type combinedPos struct {
	// runes is the offset in runes.
	runes int

	lineCol screenPos

	// Pixel coordinates
	x fixed.Int26_6
	y int

	ascent, descent fixed.Int26_6

	// runIndex tracks which run this position is within, counted each time
	// the index processes an end of run marker.
	runIndex int
	// towardOrigin tracks whether this glyph's run is progressing toward the
	// origin or away from it.
	towardOrigin bool
}

func (c combinedPos) String() string {
	return fmt.Sprintf("[combinedPos] runes: %d, lineCol: [line: %d, col: %d], x: %d, y: %d, ascent: %d, descent: %d, runIndex: %d",
		c.runes, c.lineCol.line, c.lineCol.col, c.x.Ceil(), c.y, c.ascent.Ceil(), c.descent.Ceil(), c.runIndex)
}

func (li lineInfo) String() string {
	return fmt.Sprintf("[lineInfo] xOff: %d, yOff: %d, width: %d, glyphs: %d", li.xOff.Round(), li.yOff, li.width.Ceil(), li.glyphs)
}

func newLineRange(start, end text.Glyph) lineRange {
	return lineRange{startX: start.X, startY: int(start.Y), endX: end.X, endY: int(end.Y)}
}

// makeRegion creates a text-aligned rectangle from start to end. The vertical
// dimensions of the rectangle are derived from the provided line's ascent and
// descent, and the y offset of the line's baseline is provided as y.
func makeRegion(line lineInfo, y int, start, end fixed.Int26_6) Region {
	if start > end {
		start, end = end, start
	}
	dotStart := image.Pt(start.Round(), y)
	dotEnd := image.Pt(end.Round(), y)
	return Region{
		Bounds: image.Rectangle{
			Min: dotStart.Sub(image.Point{Y: line.ascent.Ceil()}),
			Max: dotEnd.Add(image.Point{Y: line.descent.Floor()}),
		},
		Baseline: line.descent.Floor(),
	}
}

// Region describes the position and baseline of an area of interest within
// shaped text.
type Region struct {
	// Bounds is the coordinates of the bounding box relative to the containing
	// widget.
	Bounds image.Rectangle
	// Baseline is the quantity of vertical pixels between the baseline and
	// the bottom of bounds.
	Baseline int
}

// graphemeReader segments paragraphs of text into grapheme clusters.
type graphemeReader struct {
	segmenter.Segmenter
	graphemes  []int
	paragraph  []rune
	source     io.ReaderAt
	cursor     int64
	reader     *bufio.Reader
	runeOffset int
}

// SetSource configures the reader to pull from source.
func (p *graphemeReader) SetSource(source io.ReaderAt) {
	p.source = source
	p.cursor = 0
	p.reader = bufio.NewReader(p)
	p.runeOffset = 0
}

// Read exists to satisfy io.Reader. It should not be directly invoked.
func (p *graphemeReader) Read(b []byte) (int, error) {
	n, err := p.source.ReadAt(b, p.cursor)
	p.cursor += int64(n)
	return n, err
}

// next decodes one paragraph of rune data.
func (p *graphemeReader) next() ([]rune, bool) {
	p.paragraph = p.paragraph[:0]
	var err error
	var r rune
	for err == nil {
		r, _, err = p.reader.ReadRune()
		if err != nil {
			break
		}
		p.paragraph = append(p.paragraph, r)
		if r == '\n' {
			break
		}
	}
	return p.paragraph, err == nil
}

// Graphemes will return the next paragraph's grapheme cluster boundaries,
// if any. If it returns an empty slice, there is no more data (all paragraphs
// have been segmented).
func (p *graphemeReader) Graphemes() []int {
	var more bool
	p.graphemes = p.graphemes[:0]
	p.paragraph, more = p.next()
	if len(p.paragraph) == 0 && !more {
		return nil
	}
	p.Segmenter.Init(p.paragraph)
	iter := p.Segmenter.GraphemeIterator()
	if iter.Next() {
		graph := iter.Grapheme()
		p.graphemes = append(p.graphemes,
			p.runeOffset+graph.Offset,
			p.runeOffset+graph.Offset+len(graph.Text),
		)
	}
	for iter.Next() {
		graph := iter.Grapheme()
		p.graphemes = append(p.graphemes, p.runeOffset+graph.Offset+len(graph.Text))
	}
	p.runeOffset += len(p.paragraph)
	return p.graphemes
}
