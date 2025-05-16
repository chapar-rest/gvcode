package decoration

// import (
// 	"iter"

// 	"gioui.org/text"
// 	"github.com/oligo/gvcode/internal/layout"
// 	"github.com/oligo/gvcode/internal/painter"
// 	"golang.org/x/image/math/fixed"
// )

// // lineSplitter split line into RenderRun on behalf of the TextPainter.
// type decorationLineSplitter struct {
// 	current painter.RenderRun
// 	// the rune offset while iterating through the line.
// 	runeOff int
// 	// the advance offset while iterating through the line.
// 	advance   fixed.Int26_6
// 	nextGlyph func() (text.Glyph, bool)
// 	stopFunc  func()
// 	// buffered runs
// 	runs []painter.RenderRun
// }

// func (rb *decorationLineSplitter) setup(line *layout.Line) {
// 	lineIter := line.All()
// 	rb.nextGlyph, rb.stopFunc = iter.Pull(lineIter)
// 	rb.current = painter.RenderRun{}
// 	rb.advance = fixed.I(0)
// }

// func (rb *decorationLineSplitter) commitLast() {
// 	if rb.current.Size() > 0 {
// 		rb.runs = append(rb.runs, rb.current)
// 		rb.current = painter.RenderRun{
// 			Offset: rb.advance,
// 		}
// 	}
// }

// func (rb *decorationLineSplitter) Split(line *layout.Line, decorations *DecorationTree) {
// 	rb.runs = rb.runs[:0]
// 	rb.runeOff = line.RuneOff

// 	tokens := decorations.QueryRange(line.RuneOff, line.RuneOff+line.Runes)
// 	if len(tokens) == 0 {
// 		run := painter.RenderRun{
// 			Glyphs: line.GetGlyphs(0, len(line.Glyphs)),
// 			Offset: 0,
// 		}

// 		rb.runs = append(rb.runs, run)
// 		return
// 	}

// 	rb.setup(line)
// 	defer rb.stopFunc()

// 	for _, token := range tokens {
// 		// check if there is any glyphs not covered by the token and put them in
// 		// one run.
// 		start, end := token.Range()
// 		rb.readUntil(start, true)
// 		if rb.current.Size() > 0 {
// 			// no style
// 			rb.commitLast()
// 		}

// 		// next read the entire token range to the current run.
// 		rb.readUntil(end, false)
// 		if rb.current.Size() > 0 {
			

// 			bgId := token.Style.Background()
// 			fgId := token.Style.Foreground()
// 			rb.current.Fg = textTokens.GetColor(fgId)
// 			rb.current.Bg = textTokens.GetColor(bgId)
// 			textStyle := token.Style.TextStyle()
// 			if textStyle.HasStyle(textstyle.UnderlineKind) {
// 				rb.current.Underline = &UnderlineStyle{}
// 			}
// 			if textStyle.HasStyle(textstyle.SquiggleKind) {
// 				rb.current.Squiggle = &SquiggleStyle{}
// 			}
// 			if textStyle.HasStyle(textstyle.StrikethroughKind) {
// 				rb.current.Strikethrough = &StrikethroughStyle{}
// 			}
// 			if textStyle.HasStyle(textstyle.BorderKind) {
// 				rb.current.Border = &BorderStyle{}
// 			}

// 			rb.commitLast()
// 		}
// 	}

// 	if rb.current.size() > 0 {
// 		rb.commitLast()
// 	}

// }

// func (rb *decorationLineSplitter) readUntil(runeOff int, drop bool) {
// 	for rb.runeOff < runeOff {
// 		g, ok := rb.nextGlyph()
// 		if !ok {
// 			break
// 		}
// 		rb.advance += g.Advance
// 		if !drop {
// 			rb.current.Glyphs = append(rb.current.Glyphs, g)
// 		}
// 		rb.runeOff += int(g.Runes)
// 	}
// }
