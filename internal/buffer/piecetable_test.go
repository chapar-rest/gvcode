package buffer

import (
	"testing"
)

func readTableContent(pt *PieceTable) string {
	reader := PieceTableReader{PieceTable: pt}
	buf := []byte{}
	return string(reader.Text(buf))
}

func TestInsert(t *testing.T) {
	pt := NewPieceTable([]byte{})
	pt.Insert(0, "Hello, world")
	pt.Insert(6, " Go")

	if readTableContent(pt) != "Hello, Go world" {
		t.Fail()
	}

	pt = NewPieceTable([]byte("Hello, world"))
	pt.Insert(6, " Go")
	pt.Insert(6, " welcome to the")

	expected := readTableContent(pt)
	if expected != "Hello, welcome to the Go world" {
		t.Fail()
	}
}

func TestAppendInsert(t *testing.T) {
	pt := NewPieceTable([]byte{})
	pt.Insert(0, "H")
	pt.Insert(1, "e")
	pt.Insert(2, "l")
	pt.Insert(3, "l")
	pt.Insert(4, "o")

	expected := readTableContent(pt)
	if expected != "Hello" {
		t.Fail()
	}

	if pt.pieces.Length() != 1 {
		t.Fail()
	}

	pt.Insert(5, ", world")
	if pt.pieces.Length() != 2 {
		t.Fail()
	}

}

func TestUndo(t *testing.T) {
	pt := NewPieceTable([]byte(""))

	pt.Insert(0, "Hello, ")
	pt.Insert(7, "world")

	if pt.undoStack.depth() != 2 {
		t.Fail()
	}

	if pt.redoStack.depth() != 0 {
		t.Fail()
	}

	if pt.seqLength != 12 {
		t.Fail()
	}

	if pt.seqBytes != 12 {
		t.Fail()
	}

	pt.Undo()
	if pt.undoStack.depth() != 1 {
		t.Fail()
	}

	if pt.redoStack.depth() != 1 {
		t.Fail()
	}

	if pt.seqLength != 7 {
		t.Fail()
	}

	if pt.seqBytes != 7 {
		t.Fail()
	}

	expected := readTableContent(pt)
	if expected != "Hello, " {
		t.Fail()
	}

	pt.Undo()

	if pt.undoStack.depth() != 0 {
		t.Fail()
	}
	if pt.redoStack.depth() != 2 {
		t.Fail()
	}

	expected = readTableContent(pt)
	if expected != "" {
		t.Fail()
	}

}

func TestUndoRedo(t *testing.T) {
	pt := NewPieceTable([]byte(""))

	pt.Insert(0, "Hello")

	if pt.undoStack.depth() != 1 {
		t.Fail()
	}

	//runeLen, bytes :=  pt.undoStack.ranges[0].Length()

	//t.Logf("undostack range length: %d, %d", runeLen, bytes)

	if pt.redoStack.depth() != 0 {
		t.Fail()
	}

	pt.Undo()
	if pt.undoStack.depth() != 0 {
		t.Fail()
	}

	if pt.redoStack.depth() != 1 {
		t.Fail()
	}

	pt.Redo()
	if pt.undoStack.depth() != 1 {
		t.Fail()
	}

	if pt.redoStack.depth() != 0 {
		t.Fail()
	}

	// After insert or other operations, redo stack should be empty.
	pt.Insert(5, "world")
	pt.Undo()
	pt.Insert(5, "Golang")
	if pt.redoStack.depth() > 0 {
		t.Fail()
	}

}

func TestErase(t *testing.T) {
	pt := NewPieceTable([]byte(""))
	pt.Insert(0, "Hello")
	pt.Insert(5, ",world")

	// Erase start at the boundary of start piece, and end in the middle of the first piece.
	pt.Erase(0, 3)
	content := readTableContent(pt)
	if content != "lo,world" {
		t.Log("current pieces in piece table: ")
		for p := pt.pieces.Head(); p != pt.pieces.tail; p = p.next {
			t.Log(string(pt.getBuf(p.source).getTextByRange(p.byteOff, p.byteLength)))
		}

		t.Errorf("expected: lo,world, actual: %s", content)
	}

	pt.Undo()
	content = readTableContent(pt)
	if content != "Hello,world" {
		t.Log(content)
		t.Fail()
	}

	// Erase start and end in the middle of a piece
	pt.Erase(6, 8)
	content = readTableContent(pt)
	if content != "Hello,rld" {
		t.Log(content)
		t.Fail()
	}

	pt.Undo()

	// Erase start in the middle of a piece, and end in the boundary.
	pt.Erase(2, 5)
	content = readTableContent(pt)
	if content != "He,world" {
		t.Log(content)
		t.Fail()
	}

	pt.Undo()

	// Erase start and end in the boundary.
	pt.Erase(0, 5)
	content = readTableContent(pt)
	if content != ",world" {
		t.Log(content)
		t.Fail()
	}

	pt.Undo()
	// Erase start and end in the boundary.
	pt.Erase(0, 11)
	content = readTableContent(pt)
	if content != "" {
		t.Log(content)
		t.Fail()
	}
}

func TestGroupOp(t *testing.T) {
	pt := NewPieceTable([]byte(""))

	pt.GroupOp()
	batchId1 := pt.currentBatch

	{
		pt.GroupOp()
		pt.UnGroupOp()
		batchId2 := pt.currentBatch

		if batchId2 != batchId1 {
			t.Fail()
		}
	}

	pt.UnGroupOp()

	batchId3 := pt.currentBatch
	if batchId3 == batchId1 {
		t.Fail()
	}
}
