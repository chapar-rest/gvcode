package buffer

import (
	"testing"
)

func TestLineIndexInsert(t *testing.T) {
	idx := &lineIndex{}

	idx.UpdateOnInsert(0, []byte("hello\nworld"))

	if len(idx.lines) != 2 || idx.lines[0].length != 6 || idx.lines[1].length != 5 {
		t.Log(idx.lines)
		t.Fail()
	}

	// insert at the end
	idx.UpdateOnInsert(11, []byte(" one"))
	if len(idx.lines) != 2 || idx.lines[0].length != 6 || idx.lines[1].length != 9 {
		t.Log(idx.lines)
		t.Fail()
	}

	// insert in the middle of line
	idx.UpdateOnInsert(2, []byte("abc"))
	if len(idx.lines) != 2 || idx.lines[0].length != 9 || idx.lines[1].length != 9 {
		t.Log(idx.lines)
		t.Fail()
	}

	idx.UpdateOnInsert(5, []byte("\nedf"))
	if len(idx.lines) != 3 || idx.lines[0].length != 6 || idx.lines[1].length != 7 || idx.lines[2].length != 9 {
		t.Log(idx.lines)
		t.Fail()
	}

	idx.Undo()
	if len(idx.lines) != 2 || idx.lines[0].length != 9 || idx.lines[1].length != 9 {
		t.Log(idx.lines)
		t.Fail()
	}

}

func TestLineIndexDelete(t *testing.T) {
	idx := &lineIndex{}
	idx.lines = append(idx.lines,
		lineInfo{length: 4, hasLineBreak: true},
		lineInfo{length: 4, hasLineBreak: true},
		lineInfo{length: 3, hasLineBreak: false},
	)

	idx.UpdateOnDelete(0, 11)

	if len(idx.lines) != 0 {
		t.Fail()
	}

	idx.UpdateOnInsert(0, []byte("abc\nabc\nabc"))
	t.Log("lines 1: ", idx.lines)

	idx.UpdateOnDelete(5, 2)
	t.Log("lines: ", idx.lines)
	if len(idx.lines) != 3 {
		t.Log(idx.lines)
		t.Fail()
	}

}
