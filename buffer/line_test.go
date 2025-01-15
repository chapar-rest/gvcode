package buffer

import "testing"

func TestLineIndexInsertDelete(t *testing.T) {
	idx := &lineIndex{}

	printIdx := func() {
		for i := range idx.lines {
			t.Logf("line %d len: %d", i, idx.lines[i].length)
		}
	}

	idx.UpdateOnInsert(0, []byte("hello\nworld"))

	if len(idx.lines) != 2 || idx.lines[0].length != 6 || idx.lines[1].length != 5 {
		printIdx()
		t.Fail()
	}

	// insert at the end
	idx.UpdateOnInsert(11, []byte(" one"))
	if len(idx.lines) != 2 || idx.lines[0].length != 6 || idx.lines[1].length != 9 {
		printIdx()
		t.Fail()
	}

	// insert in the middle of line
	idx.UpdateOnInsert(2, []byte("abc"))
	if len(idx.lines) != 2 || idx.lines[0].length != 9 || idx.lines[1].length != 9 {
		printIdx()
		t.Fail()
	}

	idx.UpdateOnInsert(5, []byte("\nedf"))
	if len(idx.lines) != 3 || idx.lines[0].length != 6 || idx.lines[1].length != 7 || idx.lines[2].length != 9 {
		printIdx()
		t.Fail()
	}

	idx.Undo()
	if len(idx.lines) != 2 || idx.lines[0].length != 9 || idx.lines[1].length != 9 {
		// printIdx()
		t.Fail()
	}

}
