package buffer

import (
	"io"
)

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
