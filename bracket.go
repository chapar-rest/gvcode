package gvcode

import (
	"maps"
)

type bracket struct {
	r   rune
	pos int // rune offset.
}

type bracketStack struct {
	idx []bracket
}

func (s *bracketStack) push(item rune, pos int) {
	s.idx = append(s.idx, bracket{r: item, pos: pos})
}

func (s *bracketStack) pop() (rune, int) {
	if len(s.idx) == 0 {
		return 0, 0
	}

	last := s.idx[len(s.idx)-1]
	s.idx = s.idx[:len(s.idx)-1]
	return last.r, last.pos
}

func (s *bracketStack) peek() (rune, int) {
	if len(s.idx) == 0 {
		return 0, 0
	}

	last := s.idx[len(s.idx)-1]
	return last.r, last.pos
}

func (s *bracketStack) depth() int {
	return len(s.idx)
}

func (s *bracketStack) reset() {
	s.idx = s.idx[:0]
}

func rervesedBracketPairs(pairs map[rune]rune) map[rune]rune {
	dest := make(map[rune]rune)
	for k, v := range pairs {
		dest[v] = k
	}

	return dest
}

func checkBracket(pairs map[rune]rune, r rune) (_ bool, isLeft bool) {
	if _, ok := pairs[r]; ok {
		return true, true
	}

	if _, ok := rervesedBracketPairs(pairs)[r]; ok {
		return true, false
	}

	return false, false
}

func mergeMaps(sources ...map[rune]rune) map[rune]rune {
	dest := make(map[rune]rune)
	for _, src := range sources {
		maps.Copy(dest, src)
	}

	return dest
}

// NearestMatchingBrackets finds the nearest matching brackets of the caret.
func (e *textView) NearestMatchingBrackets() (left int, right int) {
	left, right = -1, -1
	start, end := e.Selection()
	if start != end {
		return
	}

	stack := &bracketStack{}
	stack.reset()

	start = min(start, e.Len())
	nearest, err := e.src.ReadRuneAt(start)
	isBracket, _ := checkBracket(e.BracketPairs, nearest)
	if err != nil || !isBracket {
		start = max(0, start-1)
		nearest, _ = e.src.ReadRuneAt(start)
	}

	if isBracket, isLeft := checkBracket(e.BracketPairs, nearest); isBracket {
		if isLeft {
			left = start
		} else {
			right = start
		}
		stack.push(nearest, start)
	}

	rtlBrackets := rervesedBracketPairs(e.BracketPairs)
	offset := start

	// find the left half.
	if left < 0 {
		for {
			offset = max(offset-1, 0)
			next, err := e.src.ReadRuneAt(offset)
			if err != nil {
				break
			}

			if br, ok := e.BracketPairs[next]; ok {
				if r, _ := stack.peek(); r == br {
					stack.pop()
					if right >= 0 && stack.depth() == 0 {
						left = offset
						break
					}
				} else {
					stack.push(next, offset)
					left = offset
					break
				}
			}

			// found a right half bracket.
			if _, ok := rtlBrackets[next]; ok {
				stack.push(next, offset)
			}

			if offset <= 0 {
				break
			}
		}
	}

	// find the right half.
	if right < 0 {
		for {
			offset = min(offset+1, e.Len())
			next, err := e.src.ReadRuneAt(offset)
			if err != nil {
				break
			}

			// found left half bracket
			if _, ok := e.BracketPairs[next]; ok {
				stack.push(next, offset)
			}

			// found a right half bracket.
			if bl, ok := rtlBrackets[next]; ok {
				if r, _ := stack.peek(); r == bl {
					stack.pop()
					if stack.depth() == 0 {
						right = offset
						break
					}
				} else {
					// Found a un-balanced bracket, drop it.
					//e.idx.push(next, offset)
				}
			}

			if offset >= e.Len() {
				break
			}

		}
	}

	return left, right
}
