package buffer

// The specific rule that resolves ambiguity when an edit happens
// exactly at a marker's position.
type MarkerBias uint8

const (
	// BiasForward sets the rule that:
	//
	// - when the marker is at the start of the edit, marker moves
	// to the end of the inserted text.
	//
	// - when the marker is at the end of the edit, it is pushed to
	// the end of the inserted text
	BiasForward = iota

	// BiasBackward sets the rule that:
	//
	// - when the marker is at the start of the edit, it keeps staying
	// at the begining.
	//
	// - when the marker is at the end of the edit, it gets pulled to
	// the start of the inserted text.
	BiasBackward
)

// Marker is a text buffer annotation that tracks a logical location
// in the buffer over time. It tries to remain logically stationary
// even when the content changes.
type Marker struct {
	// The piece reference that the marker belongs to.
	piece *piece
	// rune offset of the marker in the piece.
	offset int
	bias   MarkerBias
	valid  bool
}

// Update the marker after each editing of the buffer.
// startOffset: The start of the replaced range.
// replacedLen: The length of the text that was removed in runes.
// insertedLen: The length of the new text in runes.
func (m *Marker) Update(startOff, replacedLen, insertedLen int) {
	endOff := startOff + replacedLen
	delta := insertedLen - replacedLen

	if m.offset < startOff {
		// Marker is before the change; do nothing.
		return
	}

	if m.offset > endOff {
		// Marker is after the change; just shift it.
		m.offset += delta
		return
	}

	// Marker is at a boundary or inside the replaced range. use bias
	// to determine how to shift the offset.
	switch m.offset {
	case startOff:
		if m.bias == BiasForward {
			// Forward bias: marker moves to the end of the inserted text.
			m.offset += insertedLen
		} else {
			// Backward bias: marker stays put at the beginning.
		}
	case endOff:
		if m.bias == BiasBackward {
			// Backward bias: marker gets pulled to the start of the inserted text.
			m.offset = startOff
		} else {
			// Forward bias: marker is pushed to the end of the inserted text.
			m.offset = startOff + insertedLen
		}

	default:
		// Marker was inside the replaced range; move it to the insertion point.
		m.offset = startOff
	}

}

func (m *Marker) update(p *piece, pieceOffset int) {
	m.piece = p
	m.offset = pieceOffset
}

func newMarker(p *piece, pieceOffset int, bais MarkerBias) *Marker {
	return &Marker{
		piece:  p,
		offset: pieceOffset,
		bias:   bais,
		valid:  true,
	}
}
