// SPDX-License-Identifier: Unlicense OR MIT

package editor

import (
	"image"
	"math"
	"sort"

	"gioui.org/text"
	"golang.org/x/image/math/fixed"
)

type glyphIndex struct {
	// glyphs holds the glyphs processed.
	glyphs []text.Glyph
	// positions contain all possible caret positions, sorted by rune index.
	positions []combinedPos
	// screenLines contains metadata about the size and position of each line of
	// text on the screen.
	screenLines []lineInfo
	// lineRanges contain all line pixel coordinates in the document coordinates.
	lineRanges []lineRange

	// currentLineMin and currentLineMax track the dimensions of the line
	// that is being indexed.
	currentLineMin, currentLineMax fixed.Int26_6
	// currentLineGlyphs tracks how many glyphs are contained within the
	// line that is being indexed.
	currentLineGlyphs int
	// pos tracks attributes of the next valid cursor position within the indexed
	// text.
	pos combinedPos
	// prog tracks the current glyph text progression to detect bidi changes.
	prog text.Flags
	// clusterAdvance accumulates the advances of glyphs in a glyph cluster.
	clusterAdvance fixed.Int26_6
	// truncated indicates that the text was truncated by the shaper.
	truncated bool
	// midCluster tracks whether the next glyph processed is not the first glyph in a
	// cluster.
	midCluster bool
}

// reset prepares the index for reuse.
func (g *glyphIndex) reset() {
	g.glyphs = g.glyphs[:0]
	g.positions = g.positions[:0]
	g.screenLines = g.screenLines[:0]
	g.lineRanges = g.lineRanges[:0]
	g.lineRanges = append(g.lineRanges, lineRange{})
	g.currentLineMin = 0
	g.currentLineMax = 0
	g.currentLineGlyphs = 0
	g.pos = combinedPos{}
	g.prog = 0
	g.clusterAdvance = 0
	g.truncated = false
	g.midCluster = false
}

// incrementPosition returns the next position after pos (if any). Pos _must_ be
// an unmodified position acquired from one of the closest* methods. If eof is
// true, there was no next position.
func (g *glyphIndex) incrementPosition(pos combinedPos) (next combinedPos, eof bool) {
	candidate, index := g.closestToRune(pos.runes)
	for candidate != pos && index+1 < len(g.positions) {
		index++
		candidate = g.positions[index]
	}
	if index+1 < len(g.positions) {
		return g.positions[index+1], false
	}
	return candidate, true
}

func (g *glyphIndex) insertPosition(pos combinedPos) {
	lastIdx := len(g.positions) - 1
	if lastIdx >= 0 {
		lastPos := g.positions[lastIdx]
		if lastPos.runes == pos.runes && (lastPos.y != pos.y || (lastPos.x == pos.x)) {
			// If we insert a consecutive position with the same logical position,
			// overwrite the previous position with the new one.
			g.positions[lastIdx] = pos
			return
		}
	}
	g.positions = append(g.positions, pos)
}

func (g *glyphIndex) TrackLine(gl text.Glyph) {
	if len(g.glyphs) <= 0 {
		g.lineRanges[0] = newLineRange(gl, gl)
	}

	startParagraph := gl.Flags&text.FlagParagraphStart != 0

	if startParagraph {
		g.lineRanges = append(g.lineRanges, newLineRange(gl, gl))
	} else {
		// the line may not have a line break, so we update the end of the logical line range on
		// each coming glyph
		g.lineRanges[len(g.lineRanges)-1].endX = gl.X
		g.lineRanges[len(g.lineRanges)-1].endY = int(gl.Y)
	}
}

// Glyph indexes the provided glyph, generating text cursor positions for it.
func (g *glyphIndex) Glyph(gl text.Glyph) {
	g.glyphs = append(g.glyphs, gl)
	g.currentLineGlyphs++
	if len(g.positions) == 0 {
		// First-iteration setup.
		g.currentLineMin = math.MaxInt32
		g.currentLineMax = 0
	}
	if gl.X < g.currentLineMin {
		g.currentLineMin = gl.X
	}
	if end := gl.X + gl.Advance; end > g.currentLineMax {
		g.currentLineMax = end
	}

	needsNewLine := gl.Flags&text.FlagLineBreak != 0
	needsNewRun := gl.Flags&text.FlagRunBreak != 0
	breaksParagraph := gl.Flags&text.FlagParagraphBreak != 0
	breaksCluster := gl.Flags&text.FlagClusterBreak != 0
	// We should insert new positions if the glyph we're processing terminates
	// a glyph cluster, has nonzero runes, and is not a hard newline.
	insertPositionsWithin := breaksCluster && !breaksParagraph && gl.Runes > 0

	// Get the text progression/direction right.
	g.prog = gl.Flags & text.FlagTowardOrigin
	g.pos.towardOrigin = g.prog == text.FlagTowardOrigin
	if !g.midCluster {
		// Create the text position prior to the glyph.
		g.pos.x = gl.X
		g.pos.y = int(gl.Y)
		g.pos.ascent = gl.Ascent
		g.pos.descent = gl.Descent
		if g.pos.towardOrigin {
			g.pos.x += gl.Advance
		}
		g.insertPosition(g.pos)
	}

	g.midCluster = !breaksCluster

	if breaksParagraph {
		// Paragraph breaking clusters shouldn't have positions generated for both
		// sides of them. They're always zero-width, so doing so would
		// create two visually identical cursor positions. Just reset
		// cluster state, increment by their runes, and move on to the
		// next glyph.
		g.clusterAdvance = 0
		g.pos.runes += int(gl.Runes)
	}
	// Always track the cumulative advance added by the glyph, even if it
	// doesn't terminate a cluster itself.
	g.clusterAdvance += gl.Advance
	if insertPositionsWithin {
		// Construct the text positions _within_ gl.
		g.pos.y = int(gl.Y)
		g.pos.ascent = gl.Ascent
		g.pos.descent = gl.Descent
		width := g.clusterAdvance
		positionCount := int(gl.Runes)
		runesPerPosition := 1
		if gl.Flags&text.FlagTruncator != 0 {
			// Treat the truncator as a single unit that is either selected or not.
			positionCount = 1
			runesPerPosition = int(gl.Runes)
			g.truncated = true
		}
		perRune := width / fixed.Int26_6(positionCount)
		adjust := fixed.Int26_6(0)
		if g.pos.towardOrigin {
			// If RTL, subtract increments from the width of the cluster
			// instead of adding.
			adjust = width
			perRune = -perRune
		}
		for i := 1; i <= positionCount; i++ {
			g.pos.x = gl.X + adjust + perRune*fixed.Int26_6(i)
			g.pos.runes += runesPerPosition
			g.pos.lineCol.col += runesPerPosition
			g.insertPosition(g.pos)
		}
		g.clusterAdvance = 0
	}
	if needsNewRun {
		g.pos.runIndex++
	}
	if needsNewLine {
		g.screenLines = append(g.screenLines, lineInfo{
			xOff:    g.currentLineMin,
			yOff:    int(gl.Y),
			width:   g.currentLineMax - g.currentLineMin,
			ascent:  g.positions[len(g.positions)-1].ascent,
			descent: g.positions[len(g.positions)-1].descent,
			glyphs:  g.currentLineGlyphs,
		})

		//log.Printf("[%d] found new line: %s", len(g.screenLines), g.screenLines[len(g.screenLines)-1])

		g.pos.lineCol.line++
		g.pos.lineCol.col = 0
		g.pos.runIndex = 0
		g.currentLineMin = math.MaxInt32
		g.currentLineMax = 0
		g.currentLineGlyphs = 0
	}
}

func (g *glyphIndex) closestToRune(runeIdx int) (combinedPos, int) {
	if len(g.positions) == 0 {
		return combinedPos{}, 0
	}
	i := sort.Search(len(g.positions), func(i int) bool {
		pos := g.positions[i]
		return pos.runes >= runeIdx
	})
	if i > 0 {
		i--
	}
	closest := g.positions[i]
	closestI := i
	for ; i < len(g.positions); i++ {
		if g.positions[i].runes == runeIdx {
			return g.positions[i], i
		}
	}
	return closest, closestI
}

func (g *glyphIndex) closestToLineCol(lineCol screenPos) combinedPos {
	if len(g.positions) == 0 {
		return combinedPos{}
	}
	i := sort.Search(len(g.positions), func(i int) bool {
		pos := g.positions[i]
		return pos.lineCol.line > lineCol.line || (pos.lineCol.line == lineCol.line && pos.lineCol.col >= lineCol.col)
	})
	if i > 0 {
		i--
	}
	prior := g.positions[i]
	if i+1 >= len(g.positions) {
		return prior
	}
	next := g.positions[i+1]
	if next.lineCol != lineCol {
		return prior
	}
	return next
}

func (g *glyphIndex) closestToXY(x fixed.Int26_6, y int) combinedPos {
	if len(g.positions) == 0 {
		return combinedPos{}
	}
	i := sort.Search(len(g.positions), func(i int) bool {
		pos := g.positions[i]
		return pos.y+pos.descent.Round() >= y
	})
	// If no position was greater than the provided Y, the text is too
	// short. Return either the last position or (if there are no
	// positions) the zero position.
	if i == len(g.positions) {
		return g.positions[i-1]
	}
	first := g.positions[i]
	// Find the best X coordinate.
	closest := i
	closestDist := dist(first.x, x)
	line := first.lineCol.line
	// NOTE(whereswaldon): there isn't a simple way to accelerate this. Bidi text means that the x coordinates
	// for positions have no fixed relationship. In the future, we can consider sorting the positions
	// on a line by their x coordinate and caching that. It'll be a one-time O(nlogn) per line, but
	// subsequent uses of this function for that line become O(logn). Right now it's always O(n).
	for i := i + 1; i < len(g.positions) && g.positions[i].lineCol.line == line; i++ {
		candidate := g.positions[i]
		distance := dist(candidate.x, x)
		// If we are *really* close to the current position candidate, just choose it.
		if distance.Round() == 0 {
			return g.positions[i]
		}
		if distance < closestDist {
			closestDist = distance
			closest = i
		}
	}
	return g.positions[closest]
}

// locate returns highlight regions covering the glyphs that represent the runes in
// [startRune,endRune). If the rects parameter is non-nil, locate will use it to
// return results instead of allocating, provided that there is enough capacity.
// The returned regions have their Bounds specified relative to the provided
// viewport.
func (g *glyphIndex) locate(viewport image.Rectangle, startRune, endRune int, rects []Region) []Region {
	if startRune > endRune {
		startRune, endRune = endRune, startRune
	}
	rects = rects[:0]
	caretStart, _ := g.closestToRune(startRune)
	caretEnd, _ := g.closestToRune(endRune)

	for lineIdx := caretStart.lineCol.line; lineIdx < len(g.screenLines); lineIdx++ {
		if lineIdx > caretEnd.lineCol.line {
			break
		}
		pos := g.closestToLineCol(screenPos{line: lineIdx})
		if int(pos.y)+pos.descent.Ceil() < viewport.Min.Y {
			continue
		}
		if int(pos.y)-pos.ascent.Ceil() > viewport.Max.Y {
			break
		}
		line := g.screenLines[lineIdx]
		if lineIdx > caretStart.lineCol.line && lineIdx < caretEnd.lineCol.line {
			startX := line.xOff
			endX := startX + line.width
			// The entire line is selected.
			rects = append(rects, makeRegion(line, pos.y, startX, endX))
			continue
		}
		selectionStart := caretStart
		selectionEnd := caretEnd
		if lineIdx != caretStart.lineCol.line {
			// This line does not contain the beginning of the selection.
			selectionStart = g.closestToLineCol(screenPos{line: lineIdx})
		}
		if lineIdx != caretEnd.lineCol.line {
			// This line does not contain the end of the selection.
			selectionEnd = g.closestToLineCol(screenPos{line: lineIdx, col: math.MaxInt})
		}

		var (
			startX, endX fixed.Int26_6
			eof          bool
		)
	lineLoop:
		for !eof {
			startX = selectionStart.x
			if selectionStart.runIndex == selectionEnd.runIndex {
				// Commit selection.
				endX = selectionEnd.x
				rects = append(rects, makeRegion(line, pos.y, startX, endX))
				break
			} else {
				currentDirection := selectionStart.towardOrigin
				previous := selectionStart
			runLoop:
				for !eof {
					// Increment the start position until the next logical run.
					for startRun := selectionStart.runIndex; selectionStart.runIndex == startRun; {
						previous = selectionStart
						selectionStart, eof = g.incrementPosition(selectionStart)
						if eof {
							endX = selectionStart.x
							rects = append(rects, makeRegion(line, pos.y, startX, endX))
							break runLoop
						}
					}
					if selectionStart.towardOrigin != currentDirection {
						endX = previous.x
						rects = append(rects, makeRegion(line, pos.y, startX, endX))
						break
					}
					if selectionStart.runIndex == selectionEnd.runIndex {
						// Commit selection.
						endX = selectionEnd.x
						rects = append(rects, makeRegion(line, pos.y, startX, endX))
						break lineLoop
					}
				}
			}
		}
	}
	for i := range rects {
		rects[i].Bounds = rects[i].Bounds.Sub(viewport.Min)
	}
	return rects
}

func dist(a, b fixed.Int26_6) fixed.Int26_6 {
	if a > b {
		return a - b
	}
	return b - a
}
