package gvcode

// import (
// 	"sort"

// 	"github.com/go-text/typesetting/segmenter"
// )

// type wordOffset struct {
// 	runes int
// 	bytes int
// }

// type wordReader interface {
// 	ReadRuneAt(byteOff int64) (rune, int, error)
// }

// type uax14WordReader struct {
// 	seg segmenter.Segmenter
// }

// type wordOffIndex struct {
// 	src      wordReader
// 	offIndex []wordOffset
// }

// // indexOfRune returns the latest rune index and byte offset no later than runeIndex.
// func (w *wordOffIndex) indexOfWord(runeIndex int) wordOffset {
// 	// Initialize index.
// 	if len(w.offIndex) == 0 {
// 		w.offIndex = append(w.offIndex, wordOffset{})
// 	}

// 	i := sort.Search(len(r.offIndex), func(i int) bool {
// 		entry := r.offIndex[i]
// 		return entry.runes >= runeIndex
// 	})

// 	// Return the entry guaranteed to be less than or equal to runeIndex.
// 	if i > 0 {
// 		i--
// 	}

// 	return r.offIndex[i]
// }

// // runeOffset returns the byte offset of the source buffer's runeIndex'th rune.
// // runeIndex must be a valid rune index.
// func (r *wordOffIndex) NextWordOffset(runeIndex int, direction int) int {
// 	const runesPerIndexEntry = 50
// 	entry := r.indexOfRune(runeIndex)
// 	lastEntry := r.offIndex[len(r.offIndex)-1].runes

// 	for entry.runes < runeIndex {
// 		if entry.runes > lastEntry && entry.runes%runesPerIndexEntry == runesPerIndexEntry-1 {
// 			r.offIndex = append(r.offIndex, entry)
// 		}
// 		_, s, _ := r.src.ReadRuneAt(int64(entry.bytes))
// 		entry.bytes += s
// 		entry.runes++
// 	}

// 	return entry.bytes
// }
