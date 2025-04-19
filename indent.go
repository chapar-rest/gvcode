package gvcode

import (
	"bufio"
	"io"
	"slices"
	"strings"
)

// IndentMultiLines indent or dedent each of the selected non-empty lines with
// one tab(soft tab or hard tab). If there is now selection, the current line is
// indented or dedented.
func (e *textView) IndentMultiLines(lines []byte, linesStart, linesEnd int, dedent bool) int {
	lineReader := bufio.NewReader(strings.NewReader(string(lines)))
	newLines := strings.Builder{}
	moves := 0
	caretMoves := 0
	caretStart, caretEnd := e.Selection()
	// caret columns in runes
	_, caretCol := e.CaretPos()

	for i := 0; ; i++ {
		line, err := lineReader.ReadBytes('\n')
		if err == io.EOF && len(line) == 0 {
			break
		}

		// empty line with only the trailing line break
		if len(line) == 1 {
			newLines.Write(line)
			continue
		}

		if dedent {
			newLine := e.dedentLine(string(line))
			newLines.WriteString(newLine)
			delta := len([]rune(newLine)) - len([]rune(string(line)))
			moves += delta
			if caretEnd > caretStart {
				caretMoves = max(delta, -caretCol)
			} else {
				// capture the last line indent moves
				caretMoves = delta
			}

		} else {
			newLines.WriteString(e.Indentation() + string(line))
			moves += len([]rune(e.Indentation()))
		}
	}

	n := e.Replace(linesStart, linesEnd, newLines.String())

	if moves != 0 {
		// adjust caret positions
		if dedent {
			// When lines are dedented.
			if caretEnd < caretStart {
				e.SetCaret(caretStart+moves, caretEnd+caretMoves)
			} else {
				e.SetCaret(caretStart+caretMoves, caretEnd+moves)
			}
		} else {
			// When lines are indented, expand the end of the selection.
			if caretEnd > caretStart {
				e.SetCaret(caretStart, caretEnd+moves)
			} else {
				e.SetCaret(caretStart+moves, caretEnd)
			}
		}
	}

	return n
}

func (e *textView) dedentLine(line string) string {
	level := 0
	spaces := 0
	off := 0
	for i, r := range line {
		if r == '\t' {
			spaces = 0
			off = i
			level++
		} else if r == ' ' {
			if spaces == 0 || spaces == e.TabWidth {
				off = i
				if spaces == e.TabWidth {
					spaces = 0
				}
			}
			spaces++
			if spaces == e.TabWidth {
				level++
				continue
			}
		} else {
			// other chars
			break
		}
	}

	if spaces > 0 {
		// trim left over spaces first
		return string(slices.Delete([]rune(line), off, off+spaces))
	} else if level > 0 {
		// try to delete a single tab just before the non-spaces text.
		return string(slices.Delete([]rune(line), off, off+1))
	}

	return line
}

// IndentOnBreak insert a line break at the the current caret position, and if there is any indentation
// of the previous line, it indent the new inserted line with the same size. Furthermore, if the newline
// if between a pair of brackets, it also insert indented lines between them.
//
// This is mainly used as the line break handler when Enter or Return is pressed.
func (e *textView) IndentOnBreak(prevLine []byte, s string) int {
	indents := 0
	spaces := 0
	for _, r := range string(prevLine) {
		if r == '\t' {
			indents++
		} else if r == ' ' {
			spaces++
			if spaces == e.TabWidth {
				indents++
				spaces = 0
				continue
			}
		} else {
			// other chars
			break
		}
	}

	if indents > 0 {
		s = s + strings.Repeat(e.Indentation(), indents)
	}

	start, end := e.Selection()
	changed := e.Replace(start, end, s)
	if changed <= 0 {
		return changed
	}
	// Check if the caret is between a pair of brackets. If so we insert one more
	// indented empty line between the pair of brackets.
	changed += e.indentInsideBrackets(indents)
	return changed
}

// indentInsideBrackets checks if the caret is between two adjacent brackets pairs and insert
// indented lines between them.
func (e *textView) indentInsideBrackets(indents int) int {
	start, end := e.Selection()
	if start <= 0 || start != end {
		return 0
	}

	indentation := e.Indentation()
	moves := indents * len([]rune(indentation))

	leftRune, err1 := e.src.ReadRuneAt(start - 2 - moves) // offset to index
	rightRune, err2 := e.src.ReadRuneAt(min(start, e.Len()))

	if err1 != nil || err2 != nil {
		return 0
	}

	insideBrackets := rightRune == e.BracketPairs[leftRune]
	if insideBrackets {
		// move to the left side of the line break.
		e.MoveCaret(-moves, -moves)
		// Add one more line and indent one more level.
		return e.Replace(start, end, strings.Repeat(indentation, indents+1)+"\n")
	}

	return 0
}


// func (e *autoIndentHandler) dedentRightBrackets(ke key.EditEvent) bool {
// 	opening, ok := rtlBracketPairs[ke.Text]
// 	if !ok {
// 		return false
// 	}
