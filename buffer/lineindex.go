package buffer

import (
	"slices"
	"unicode/utf8"
)

const (
	lineBreak = '\n'
)

type lineInfo struct {
	length       int
	hasLineBreak bool
}

type lineOp struct {
	action    action
	runeIndex int
	length    int
	lines     []lineInfo
}

type lineOpStack struct {
	ops []*lineOp
}

// lineIndex manages a line index for the text sequence using an incremental manner.
// It also provides its own undo/redo stack.
type lineIndex struct {
	// Index of the slice saves the continuous line number starting from zero.
	// The value contains the rune length of the line.
	lines []lineInfo
	// undo stack & redo stack used to update the index after the piece range is undone or redone.
	undo lineOpStack
	redo lineOpStack
}

func (li *lineIndex) UpdateOnInsert(runeIndex int, text []byte) {
	li.redo.clear()
	newLines := li.parseLine(text)
	li.applyInsert(runeIndex, newLines)

	op := &lineOp{
		action:    actionInsert,
		runeIndex: runeIndex,
		length:    utf8.RuneCountInString(string(text)),
		lines:     newLines,
	}

	li.undo.push(op)
}

func (li *lineIndex) UpdateOnDelete(runeIndex int, length int) {
	li.redo.clear()
	removedLines := li.applyDelete(runeIndex, length)

	op := &lineOp{
		action:    actionErase,
		runeIndex: runeIndex,
		length:    length,
		lines:     removedLines,
	}
	li.undo.push(op)

}

func (li *lineIndex) Undo() {
	src := li.undo
	dest := li.redo

	op := src.pop()
	if op == nil {
		return
	}

	if op.action == actionInsert {
		li.applyDelete(op.runeIndex, op.length)
	} else if op.action == actionErase {
		li.applyInsert(op.runeIndex, op.lines)
	}

	dest.push(op)
}

func (li *lineIndex) Redo() {
	src := li.redo
	dest := li.undo

	op := src.pop()
	if op == nil {
		return
	}

	if op.action == actionInsert {
		li.applyInsert(op.runeIndex, op.lines)
	} else if op.action == actionErase {
		li.applyDelete(op.runeIndex, op.length)
	}

	dest.push(op)
}

func (li *lineIndex) applyInsert(runeIndex int, newLines []lineInfo) {
	if len(newLines) <= 0 {
		return
	}

	var currentRuneCount int
	var insertionIndex int

	// Locate the insertion point in the existing line index
	for i, line := range li.lines {
		if currentRuneCount+line.length > runeIndex {
			insertionIndex = i
			break
		}
		currentRuneCount += line.length
	}

	// Check if we have found a insertion point.
	if currentRuneCount > 0 && insertionIndex == 0 {
		// No insertion found, increase the index to the next.
		insertionIndex = len(li.lines)
	}

	// Split the line at the insertion point if necessary
	if insertionIndex < len(li.lines) {
		line := li.lines[insertionIndex]

		// Prepare for splitting and merging
		splitLeft := runeIndex - currentRuneCount

		if splitLeft >= 0 {
			if len(newLines) == 1 && !newLines[0].hasLineBreak {
				// just merge the new fragment.
				li.lines[insertionIndex].length += newLines[0].length
			} else {
				// Create a left part from the split
				leftPart := lineInfo{length: splitLeft + newLines[0].length, hasLineBreak: newLines[0].hasLineBreak}
				li.lines[insertionIndex] = leftPart
			}

			newLines = newLines[1:]

			var rightPart lineInfo
			if len(newLines) > 0 {
				lastLine := newLines[len(newLines)-1]
				if !lastLine.hasLineBreak {
					rightPart = lineInfo{length: line.length - splitLeft + lastLine.length, hasLineBreak: line.hasLineBreak}
					newLines = newLines[:len(newLines)-1]
				} else {
					rightPart = lineInfo{length: line.length - splitLeft, hasLineBreak: line.hasLineBreak}
				}

				li.lines = slices.Insert(li.lines, insertionIndex+1, rightPart)
			}

			if len(newLines) > 0 {
				li.lines = slices.Insert(li.lines, insertionIndex+1, newLines...)
			}
		}

		return
	}

	// If the last line does not have a line break, merge the first new line with it.
	if len(li.lines) > 0 {
		lastLine := li.lines[len(li.lines)-1]
		if !lastLine.hasLineBreak {
			lastLine.length += newLines[0].length
			lastLine.hasLineBreak = newLines[0].hasLineBreak
			li.lines[len(li.lines)-1] = lastLine
			newLines = newLines[:len(newLines)-1]
		}
	}

	// Append the remaining lines
	li.lines = append(li.lines, newLines...)
}

func (li *lineIndex) applyDelete(runeIndex int, length int) []lineInfo {
	var currentRuneCount int
	var startIndex, endIndex int
	var removedLines []lineInfo

	// Locate the starting and ending indices of the deletion
	for i, line := range li.lines {
		if currentRuneCount+line.length > runeIndex {
			startIndex = i
			break
		}
		currentRuneCount += line.length
	}

	for i, line := range li.lines[startIndex:] {
		if currentRuneCount+line.length >= runeIndex+length {
			endIndex = startIndex + i
			break
		}
		currentRuneCount += line.length
	}

	// Handle the splitting of the starting line
	//startLine := li.lines[startIndex]
	splitLeft := runeIndex - currentRuneCount
	if splitLeft > 0 {
		leftPart := lineInfo{length: splitLeft, hasLineBreak: false}
		li.lines[startIndex] = leftPart
	} else {
		li.lines = append(li.lines[:startIndex], li.lines[startIndex+1:]...)
		endIndex--
	}

	// Handle the splitting of the ending line
	if endIndex < len(li.lines) {
		endLine := li.lines[endIndex]
		splitRight := (runeIndex + length) - currentRuneCount
		if splitRight < endLine.length {
			rightPart := lineInfo{length: endLine.length - splitRight, hasLineBreak: endLine.hasLineBreak}
			li.lines[endIndex] = rightPart
		} else {
			li.lines = append(li.lines[:endIndex], li.lines[endIndex+1:]...)
		}
	}

	// Capture the removed lines
	removedLines = li.lines[startIndex:endIndex]
	li.lines = append(li.lines[:startIndex], li.lines[endIndex:]...)

	return removedLines
}

// func (li *lineIndex) applyDelete(runeIndex int, length int) []lineInfo {
// 	removedLines := make([]lineInfo, 0)

// 	// find the nearst line the delete starts.
// 	lineLen := 0
// 	for i := range li.lines {
// 		lineLen += li.lines[i].length
// 		if lineLen > runeIndex && length > 0 {
// 			deleted := 0
// 			if runeIndex-(lineLen-li.lines[i].length) < 0 {
// 				deleted = min(length, li.lines[i].length-(lineLen-runeIndex))
// 			} else {
// 				deleted = min(length, li.lines[i].length)
// 			}
// 			li.lines[i].length -= deleted
// 			removedLines = append(removedLines, lineInfo{length: deleted, hasLineBreak: li.lines[i].length <= 0})
// 			length -= deleted
// 		}

// 		if length <= 0 {
// 			break
// 		}
// 	}

// 	// Line length may decrease to zero, should be removed from the index.
// 	li.lines = slices.DeleteFunc(li.lines, func(line lineInfo) bool {
// 		return line.length <= 0
// 	})

// 	return removedLines
// }

func (li *lineIndex) parseLine(text []byte) []lineInfo {
	var lines []lineInfo

	n := 0
	for _, c := range string(text) {
		n++
		if c == lineBreak {
			lines = append(lines, lineInfo{length: n, hasLineBreak: true})
			n = 0
		}
	}

	// The remaining bytes that don't end with a line break.
	if n > 0 {
		lines = append(lines, lineInfo{length: n})
	}

	return lines
}

func (li *lineIndex) Lines() []lineInfo {
	return li.lines
}

func (s *lineOpStack) push(op *lineOp) {
	s.ops = append(s.ops, op)
}

func (s *lineOpStack) pop() *lineOp {
	if len(s.ops) <= 0 {
		return nil
	}

	op := s.ops[len(s.ops)-1]
	s.ops = s.ops[:len(s.ops)-1]
	return op
}

func (s *lineOpStack) clear() {
	s.ops = s.ops[:0]
}
