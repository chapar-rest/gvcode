package buffer

// piece is a single piece of text in the piece table.
// We use doubly linked list to represent a piece table here.
type piece struct {
	next *piece
	prev *piece

	// offset is the rune offset in the buffer.
	offset int
	// length is the length of text of the piece covers.
	length int
	// byte offset in the buffer.
	byteOff int
	// byte length of the text.
	byteLength int
	// source specifies which buffer this piece point to.
	source bufSrc
}

// Use sentinel nodes to be used as head and tail, as pointed out in https://www.catch22.net/tuts/neatpad/piece-chains/.
type pieceList struct {
	head, tail *piece
	// cached piece for rapid offset query.
	lastPiece *piece
	// offset in the sequence of the last piece.
	lastPieceOff int
}

// A piece-range effectively represents the range of pieces affected by an operation on the sequence.
type pieceRange struct {
	first    *piece
	last     *piece
	boundary bool

	// The sequence length in rune before the time the operation occurs.
	seqLength int
	// The sequence byte size at before time the operation occurs.
	seqBytes int
	// the operation location in the sequence.
	runeIndex int
}

func newPieceList() *pieceList {
	p := &pieceList{
		head: &piece{},
		tail: &piece{},
	}
	p.head.next = p.tail
	p.tail.prev = p.head

	return p
}

func (pl *pieceList) Head() *piece {
	return pl.head.next
}

func (pl *pieceList) Tail() *piece {
	return pl.tail.prev
}

func (pl *pieceList) InsertBefore(existing *piece, newPiece *piece) {
	newPiece.next = existing
	newPiece.prev = existing.prev
	existing.prev.next = newPiece
	existing.prev = newPiece
}

func (pl *pieceList) InsertAfter(existing *piece, newPiece *piece) {
	newPiece.prev = existing
	newPiece.next = existing.next
	existing.next.prev = newPiece
	existing.next = newPiece
}

func (pl *pieceList) InsertBeforeTail(newPiece *piece) {
	pl.InsertBefore(pl.tail, newPiece)
}

// findPiece finds a piece by a runeIndex in the sequence/document, returning
// the found piece and it rune offset in the found piece.
func (pl *pieceList) FindPiece(runeIndex int) (p *piece, offset int) {
	if runeIndex <= 0 {
		return pl.head.next, 0
	}

	pieceOff := 0
	for n := pl.head.next; n != nil; n = n.next {
		pieceOff += n.length
		if pieceOff > runeIndex {
			p = n
			break
		}
	}

	if p == nil {
		p = pl.tail
	}

	offset = runeIndex - (pieceOff - p.length)
	return
}

// Remove a piece from the chain.
func (pl *pieceList) Remove(piece *piece) {
	if piece == nil || piece == pl.head || piece == pl.tail {
		return
	}

	piece.prev.next = piece.next
	piece.next.prev = piece.prev
}

// Length returns total pieces of the chain
func (pl *pieceList) Length() int {
	t := 0
	for n := pl.head.next; n != pl.tail; n = n.next {
		t++
	}

	return t
}

// create a new piece range by providing the current sequence length, and the operation location in rune index.
func newUndoPieceRange(seqLength, seqBytes int, runeIndex int) *pieceRange {
	return &pieceRange{
		seqLength: seqLength,
		seqBytes:  seqBytes,
		runeIndex: runeIndex,
	}
}

func newPieceRange() *pieceRange {
	return &pieceRange{}
}

// AsBoundary turns the pieceRange to a boundary range by linking its first to the prev node of target,
// and the last ndoe as target.
func (p *pieceRange) AsBoundary(target *piece) {
	p.first = target.prev
	p.last = target
	p.boundary = true
}

func (p *pieceRange) Length() int {
	if p.first == nil || p.boundary {
		return 0
	}

	len := 0
	for n := p.first; n != p.last.next; n = n.next {
		len += n.length
	}

	return len
}

func (p *pieceRange) Append(piece *piece) {
	if piece == nil {
		return
	}

	if p.first == nil {
		// first time insert of a piece
		p.first = piece
	} else {
		p.last.next = piece
		piece.prev = p.last
	}

	p.last = piece
	p.boundary = false
}

func (p *pieceRange) Swap(dest *pieceRange) {
	if p.boundary {
		if !dest.boundary {
			p.first.next = dest.first
			p.last.prev = dest.last
			dest.first.prev = p.first
			dest.last.next = p.last
		}
	} else {
		if dest.boundary {
			p.first.prev.next = p.last.next
			p.last.next.prev = p.first.prev
		} else {
			p.first.prev.next = dest.first
			p.last.next.prev = dest.last
			dest.first.prev = p.first.prev
			dest.last.next = p.last.next
		}
	}
}

// Restore the saved pieces in undo/redo stack to the main list.
func (p *pieceRange) Restore() {
	if p.boundary {
		first := p.first.next
		last := p.last.prev

		// unlink the pieces from the list
		p.first.next = p.last
		p.last.prev = p.first

		// store the removed range
		p.first = first
		p.last = last
		p.boundary = false
	} else {
		first := p.first.prev
		last := p.last.next

		// empty range
		if first.next == last {
			// move the old range back to the empty region.
			first.next = p.first
			last.prev = p.last
			// store the removed range
			p.first = first
			p.last = last
			p.boundary = true
		} else {
			// replacing a range of pieces in the list.

			// find the range that is currently in the list
			first := first.next
			last := last.prev

			// unlink
			first.prev.next = p.first
			last.next.prev = p.last

			// store
			p.first = first
			p.last = last
			p.boundary = false

		}
	}
}

// void sequence::restore_spanrange (span_range *range, bool undo_or_redo)
// {
// 	if(range->boundary)
// 	{
// 		span *first = range->first->next;
// 		span *last  = range->last->prev;

// 		// unlink spans from main list
// 		range->first->next = range->last;
// 		range->last->prev  = range->first;

// 		// store the span range we just removed
// 		range->first = first;
// 		range->last  = last;
// 		range->boundary = false;
// 	}
// 	else
// 	{
// 		span *first = range->first->prev;
// 		span *last  = range->last->next;

// 		// are we moving spans into an "empty" region?
// 		// (i.e. inbetween two adjacent spans)
// 		if(first->next == last)
// 		{
// 			// move the old spans back into the empty region
// 			first->next = range->first;
// 			last->prev  = range->last;

// 			// store the span range we just removed
// 			range->first  = first;
// 			range->last   = last;
// 			range->boundary  = true;
// 		}
// 		// we are replacing a range of spans in the list,
// 		// so swap the spans in the list with the one's in our "undo" event
// 		else
// 		{
// 			// find the span range that is currently in the list
// 			first = first->next;
// 			last  = last->prev;

// 			// unlink the the spans from the main list
// 			first->prev->next = range->first;
// 			last->next->prev  = range->last;

// 			// store the span range we just removed
// 			range->first = first;
// 			range->last  = last;
// 			range->boundary = false;
// 		}
// 	}

// 	// update the 'sequence length' and 'quicksave' states
// 	std::swap(range->sequence_length,    sequence_length);
// 	std::swap(range->quicksave,			 can_quicksave);

// 	undoredo_index	= range->index;

// 	if(range->act == action_erase && undo_or_redo == true ||
// 		range->act != action_erase && undo_or_redo == false)
// 	{
// 		undoredo_length = range->length;
// 	}
// 	else
// 	{
// 		undoredo_length = 0;
// 	}
// }
