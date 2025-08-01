package buffer

import "io"

// TextSource provides data for editor.
//
// Basic editing operations, such as insert, delete, replace,
// undo/redo are supported. If used with GroupOp and UnGroupOp,
// the undo and redo operations can be batched.
type TextSource interface {
	io.ReaderAt

	// ReadRuneAt reads the rune starting at the given rune offset, if any.
	ReadRuneAt(runeOff int) (rune, error)

	// RuneOffset returns the byte offset for the rune at position runeIndex.
	RuneOffset(runeIndex int) int

	// Lines returns the total number of lines/paragraphs of the source.
	Lines() int

	// Len is the length of the editor contents, in runes.
	Len() int

	// Size returns the size of the editor content in bytes.
	Size() int

	// SetText reset the buffer and replace the content of the buffer with the provided text.
	SetText(text []byte)

	// Replace replace text from startOff to endOff(exclusive) with text.
	Replace(startOff, endOff int, text string) bool

	// CreateMarker adds a new marker at position runeOff, with the specified bais. A bais
	// controlls how the markers move when the insertion/deletion happens at the boundary location
	// of the marker.
	CreateMarker(runeOff int, bais MarkerBias) *Marker
	// GetMarkerOffset returns the rune offset of the marker in the document.
	GetMarkerOffset(marker *Marker) int
	// RemoveMarker removes a marker from the text source.
	RemoveMarker(m *Marker)

	// Undo the last insert, erase, or replace, or a group of operations.
	// It returns all the cursor positions after undo.
	Undo() ([]CursorPos, bool)
	// Redo the last insert, erase, or replace, or a group of operations.
	// It returns all the cursor positions after undo.
	Redo() ([]CursorPos, bool)

	// Group operations such as insert, earase or replace in a batch.
	// Nested call share the same single batch.
	GroupOp()

	// Ungroup a batch. Latter insert, earase or replace operations outside of
	// a group is not batched.
	UnGroupOp()

	// Changed report whether the contents have changed since the last call to Changed.
	Changed() bool
}

type TextReader interface {
	io.Seeker
	io.Reader
	//ReadAll returns the contents of the editor.
	ReadAll(buf []byte) []byte
}
