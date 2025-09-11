package buffer

import (
	"fmt"
	"testing"
)

func readTableContent(pt *PieceTable) string {
	reader := NewReader(pt)
	buf := []byte{}
	return string(reader.ReadAll(buf))
}

func TestReplace(t *testing.T) {
	pt := NewPieceTable([]byte{})
	pt.Replace(0, 0, "Hello, world")
	pt.Replace(6, 6, " Go")

	if readTableContent(pt) != "Hello, Go world" {
		t.Fail()
	}

	pt = NewPieceTable([]byte("Hello, world"))
	pt.Replace(6, 6, " Go")
	pt.Replace(6, 6, " welcome to the")

	expected := readTableContent(pt)
	if expected != "Hello, welcome to the Go world" {
		t.Fail()
	}
}

func TestAppendReplace(t *testing.T) {
	SetDebug(true)
	pt := NewPieceTable([]byte{})
	pt.Replace(0, 0, "H")
	pt.Replace(1, 1, "e")
	pt.Replace(2, 2, "l")
	pt.Replace(3, 3, "l")
	pt.Replace(4, 4, "o")

	expected := readTableContent(pt)
	if expected != "Hello" {
		t.Logf("expected: %s", expected)
		t.Fail()
	}

	if pt.pieces.Length() != 1 {
		t.Logf("expected length: %d, actual: %d", 1, pt.pieces.Length())
		t.Fail()
	}

	pt.Replace(5, 5, ", world")
	if pt.pieces.Length() != 2 {
		t.Logf("expected length: %d, actual: %d", 2, pt.pieces.Length())
		t.Fail()
	}

}

func TestUndo(t *testing.T) {
	pt := NewPieceTable([]byte(""))

	pt.Replace(0, 0, "Hello, ")
	pt.Replace(7, 7, "world")

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

	pt.Replace(0, 0, "Hello")

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
	pt.Replace(5, 5, "world")
	pt.Undo()
	pt.Replace(5, 5, "Golang")
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
		pt.Replace(0, 0, "Hello")
		pt.Replace(5, 5, ",world")

		t.Run(tc.desc, func(t *testing.T) {
			pt.Replace(tc.input[0], tc.input[1], "")
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

func TestMarkerOnInsert(t *testing.T) {
	setup := func(bais MarkerBias, markerPos int) (*PieceTable, *Marker) {
		pt := NewPieceTable([]byte("hello,world"))
		marker, _ := pt.CreateMarker(markerPos, bais)

		if marker.Offset() != markerPos {
			t.Logf("initOffset: %d", marker.Offset())
			t.FailNow()
		}
		return pt, marker
	}

	testcases := []struct {
		insertOffset     int
		bais             MarkerBias
		wantMarkerOffset int
		marker           int
	}{
		{
			insertOffset:     4,
			bais:             BiasForward, // any value is ok
			wantMarkerOffset: 12,
			marker:           6,
		},
		{
			insertOffset:     4,
			bais:             BiasBackward, // any value is ok
			wantMarkerOffset: 12,
			marker:           6,
		},
		{
			insertOffset:     6,
			bais:             BiasForward,
			wantMarkerOffset: 12,
			marker:           6,
		},

		{
			insertOffset:     6,
			bais:             BiasBackward,
			wantMarkerOffset: 6,
			marker:           6,
		},

		{
			insertOffset:     7,
			bais:             BiasForward, // any value is ok
			wantMarkerOffset: 6,
			marker:           6,
		},
		{
			insertOffset:     7,
			bais:             BiasBackward, // any value is ok
			wantMarkerOffset: 6,
			marker:           6,
		},
		{
			insertOffset:     0,
			bais:             BiasBackward,
			wantMarkerOffset: 0,
			marker:           0,
		},
		{
			insertOffset:     0,
			bais:             BiasForward,
			wantMarkerOffset: 6,
			marker:           0,
		},
	}

	for idx, tc := range testcases {
		t.Run(fmt.Sprintf("%d-offset:%d", idx, tc.insertOffset), func(t *testing.T) {
			pt, marker := setup(tc.bais, tc.marker)
			pt.Replace(tc.insertOffset, tc.insertOffset, "golang")

			newOffset := marker.Offset()
			if newOffset != tc.wantMarkerOffset {
				t.Logf("newOffset: %d, pt: %s", newOffset, readTableContent(pt))
				t.Fail()
			}
		})
	}

}

func TestMarkerOnErase(t *testing.T) {
	setup := func(markerPos int) (*PieceTable, *Marker) {
		pt := NewPieceTable([]byte(""))
		pt.Replace(0, 0, "Hello,")
		pt.Replace(6, 6, "golang")
		pt.Replace(12, 12, " world")
		marker, _ := pt.CreateMarker(markerPos, BiasBackward)

		initOffset := marker.Offset()
		if initOffset != markerPos {
			t.Logf("initOffset: %d", initOffset)
			t.FailNow()
		}
		return pt, marker
	}

	testcases := []struct {
		eraseRange       []int
		markerPos        int
		wantMarkerOffset int
	}{
		{
			eraseRange:       []int{0, 2},
			markerPos:        3,
			wantMarkerOffset: 1,
		},
		{
			eraseRange:       []int{0, 2},
			markerPos:        2,
			wantMarkerOffset: 0,
		},

		{
			eraseRange:       []int{1, 2},
			markerPos:        1,
			wantMarkerOffset: 1,
		},

		{
			eraseRange:       []int{4, 6},
			markerPos:        3,
			wantMarkerOffset: 3,
		},
		{
			eraseRange:       []int{4, 6},
			markerPos:        5,
			wantMarkerOffset: 4,
		},

		{
			eraseRange:       []int{6, 12},
			markerPos:        7,
			wantMarkerOffset: 6,
		},

		{
			eraseRange:       []int{5, 13},
			markerPos:        14,
			wantMarkerOffset: 6,
		},
	}

	for idx, tc := range testcases {
		t.Run(fmt.Sprintf("%d-offset:%v", idx, tc.eraseRange), func(t *testing.T) {
			pt, marker := setup(tc.markerPos)
			pt.Replace(tc.eraseRange[0], tc.eraseRange[1], "")

			newOffset := marker.Offset()
			if newOffset != tc.wantMarkerOffset {
				t.Logf("expectedOffset: %d, actualOffset: %d, pt: %s", tc.wantMarkerOffset, newOffset, readTableContent(pt))
				t.Fail()
			}
		})
	}
}
