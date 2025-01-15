package buffer

import (
	"io"
	"unicode/utf8"
)

var _ TextSource = (*PieceTableReader)(nil)

// PieceTableReader implements a [TextSource].
type PieceTableReader struct {
	*PieceTable

	lastPiece  *piece
	seekCursor int64
}

// ReadAt implements [io.ReaderAt].
func (r *PieceTableReader) ReadAt(p []byte, offset int64) (total int, err error) {
	if len(p) == 0 {
		return 0, nil
	}
	if offset >= int64(r.seqBytes) {
		return 0, io.EOF
	}

	var expected = len(p)
	var bytes int64
	for n := r.pieces.Head(); n != r.pieces.tail; n = n.next {
		bytes += int64(n.byteLength)

		if bytes > offset {
			fragment := r.getBuf(n.source).getTextByRange(
				n.byteOff+n.byteLength-int(bytes-offset), // calculate the offset in the source buffer.
				int(bytes-offset))

			n := copy(p, fragment)
			p = p[n:]
			total += n
			offset += int64(n)

			if total >= expected {
				break
			}
		}

	}

	if total < expected {
		err = io.EOF
	}

	return
}

// Seek implements [io.Seeker].
func (r *PieceTableReader) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		r.seekCursor = offset
	case io.SeekCurrent:
		r.seekCursor += offset
	case io.SeekEnd:
		r.seekCursor = int64(r.seqBytes) + offset
	}
	return r.seekCursor, nil
}

// Read implements [io.Reader].
func (r *PieceTableReader) Read(p []byte) (int, error) {
	n, err := r.ReadAt(p, r.seekCursor)
	r.seekCursor += int64(n)
	return n, err
}

func (r *PieceTableReader) Text(buf []byte) []byte {
	if cap(buf) < int(r.seqBytes) {
		buf = make([]byte, r.seqBytes)
	}
	buf = buf[:r.seqBytes]
	r.Seek(0, io.SeekStart)
	n, _ := io.ReadFull(r, buf)
	buf = buf[:n]
	return buf
}

// RuneOffset returns the byte offset for the rune at position runeIndex.
func (r *PieceTableReader) RuneOffset(runeIndex int) int {
	if r.seqLength == 0 {
		return 0
	}

	if runeIndex >= r.seqLength {
		return r.seqBytes - 1
	}

	var bytes int
	var runes int

	for n := r.pieces.Head(); n != r.pieces.tail; n = n.next {
		if runes+n.length > runeIndex {
			return bytes + r.getBuf(n.source).bytesForRange(n.offset, runeIndex-runes)
		}

		bytes += n.byteLength
		runes += n.length

	}

	return bytes
}

// ReadRuneAt reads the rune starting at the given byte offset, if any.
func (r *PieceTableReader) ReadRuneAt(off int64) (rune, int, error) {
	var buf [utf8.UTFMax]byte
	b := buf[:]
	n, err := r.ReadAt(b, off)
	b = b[:n]
	c, s := utf8.DecodeRune(b)
	return c, s, err
}

// ReadRuneAt reads the run prior to the given byte offset, if any.
func (r *PieceTableReader) ReadRuneBefore(off int64) (rune, int, error) {
	var buf [utf8.UTFMax]byte
	b := buf[:]
	if off < utf8.UTFMax {
		b = b[:off]
		off = 0
	} else {
		off -= utf8.UTFMax
	}
	n, err := r.ReadAt(b, off)
	b = b[:n]
	c, s := utf8.DecodeLastRune(b)
	return c, s, err
}

func NewTextSource() *PieceTableReader {
	return &PieceTableReader{
		PieceTable: NewPieceTable([]byte("")),
	}
}
