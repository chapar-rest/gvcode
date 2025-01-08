package buffer

import (
	"testing"
)

func TestInsert(t *testing.T) {
	pt := NewPieceTable([]byte{})
	reader := PieceTableReader{PieceTable: pt}
	pt.Insert(0, "Hello, world")
	pt.Insert(6, " Go")

	buf := []byte{}

	if string(reader.Text(buf)) != "Hello, Go world" {
		t.Fail()
	}

	pt = NewPieceTable([]byte("Hello, world"))
	reader = PieceTableReader{PieceTable: pt}
	pt.Insert(6, " Go")
	pt.Insert(6, " welcome to the")

	buf = buf[:0]

	expected := string(reader.Text(buf))
	if expected != "Hello, welcome to the Go world" {
		t.Fail()
	}
}

func TestAppendInsert(t *testing.T) {
	pt := NewPieceTable([]byte{})
	reader := PieceTableReader{PieceTable: pt}
	pt.Insert(0, "H")
	pt.Insert(1, "e")
	pt.Insert(2, "l")
	pt.Insert(3, "l")
	pt.Insert(4, "o")

	buf := []byte{}
	expected := string(reader.Text(buf))
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
	reader := PieceTableReader{PieceTable: pt}

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

	buf := []byte{}
	expected := string(reader.Text(buf))
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

	buf = buf[:0]
	expected = string(reader.Text(buf))
	if expected != "" {
		t.Fail()
	}

}
