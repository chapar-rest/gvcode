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
	cases := []struct {
		desc  string
		input []int
		want  struct {
			content string
			bytes   int
		}
	}{
		{
			desc:  "Erase start at the boundary of start piece, and end in the middle of the first piece.",
			input: []int{0, 3},
			want: struct {
				content string
				bytes   int
			}{content: "lo,world", bytes: 8},
		},
		{
			desc:  "Erase start and end in the middle of a piece",
			input: []int{6, 8},
			want: struct {
				content string
				bytes   int
			}{
				content: "Hello,rld",
				bytes:   9,
			},
		},
		{
			desc:  "Erase start and end in the middle of two pieces",
			input: []int{4, 6},
			want: struct {
				content string
				bytes   int
			}{
				content: "Hellworld",
				bytes:   9,
			},
		},
		{
			desc:  "Erase start in the middle of a piece, and end in the boundary.",
			input: []int{2, 5},
			want: struct {
				content string
				bytes   int
			}{
				content: "He,world",
				bytes:   8,
			},
		},
		{
			desc:  "Erase start and end in the boundary.",
			input: []int{0, 5},
			want: struct {
				content string
				bytes   int
			}{
				content: ",world",
				bytes:   6,
			},
		},
		{
			desc:  "Erase all.",
			input: []int{0, 11},
			want: struct {
				content string
				bytes   int
			}{
				content: "",
				bytes:   0,
			},
		},
	}

	for _, tc := range cases {
		pt := NewPieceTable([]byte(""))
		pt.Insert(0, "Hello")
		pt.Insert(5, ",world")

		t.Run(tc.desc, func(t *testing.T) {
			pt.Erase(tc.input[0], tc.input[1])
			if ans := readTableContent(pt); ans != tc.want.content || pt.seqBytes != tc.want.bytes {
				t.Errorf("got content: %s, want content: %s; got bytes: %d, want bytes: %d", ans, tc.want.content, pt.seqBytes, tc.want.bytes)
			}
		})
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
