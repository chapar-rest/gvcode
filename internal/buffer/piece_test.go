package buffer

import "testing"

func TestBoundaryPieceRangeSwap(t *testing.T) {
	list := newPieceList()
	old, _, _ := list.FindPiece(0)
	//t.Errorf("old: %p, prev: %p", old, old.prev)

	rng := &pieceRange{}
	rng.AsBoundary(old)
	//t.Logf("rng.first: %p, prev: %p, rng.last: %p, ", rng.first, old.prev, rng.last)

	if rng.first != old.prev || rng.last != old {
		t.Error("set boundary range failed")
	}

	newRng := &pieceRange{}
	newPiece := &piece{
		source:     modify,
		offset:     0,
		length:     5,
		byteOff:    0,
		byteLength: 5,
	}
	newRng.Append(newPiece)

	oldPrev := old.prev

	rng.Swap(newRng)

	if list.Length() != 1 {
		t.Error("boundary pieceRange swap failed")
	}

	if rng.first != oldPrev || rng.last != old {
		t.Errorf("swap boundary range failed: exptected: [first: %p, last: %p], actual: [first: %p, last: %p]", oldPrev, old, rng.first, rng.last)
	}

}

func TestNormalToBoundaryPieceRangeSwap(t *testing.T) {
	list := newPieceList()
	old := &piece{
		source:     modify,
		offset:     0,
		length:     5,
		byteOff:    0,
		byteLength: 5,
	}
	list.Append(old)

	rng := &pieceRange{}
	rng.Append(old)

	if rng.first != old || rng.last != old {
		t.Error("set range failed")
	}

	newRng := &pieceRange{}
	newRng.AsBoundary(list.tail)

	rng.Swap(newRng)

	if list.Length() != 0 {
		t.Error("boundary pieceRange swap failed")
	}

	if rng.first.prev.next != rng.last.next || rng.last.next.prev != rng.first.prev {
		t.Errorf("invalid boundary range")
	}

}

func TestNormalToNormalPieceRangeSwap(t *testing.T) {
	list := newPieceList()
	old := &piece{
		source:     modify,
		offset:     0,
		length:     5,
		byteOff:    0,
		byteLength: 5,
	}
	list.Append(old)

	rng := &pieceRange{}
	rng.Append(old)
	//t.Logf("rng.first: %p, prev: %p, rng.last: %p, ", rng.first, old.prev, rng.last)

	if rng.first != old || rng.last != old {
		t.Error("set range failed")
	}

	newRng := &pieceRange{}
	newRng.Append(&piece{
		source:     modify,
		offset:     5,
		length:     2,
		byteOff:    5,
		byteLength: 2,
	})
	newRng.Append(&piece{
		source:     modify,
		offset:     7,
		length:     2,
		byteOff:    7,
		byteLength: 2,
	})

	rng.Swap(newRng)

	if list.Length() != 2 {
		t.Error("boundary pieceRange swap failed")
	}

	if rng.first.prev.next != newRng.first || rng.last.next.prev != newRng.last {
		t.Errorf("invalid boundary range")
	}

}
