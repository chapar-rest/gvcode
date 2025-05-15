package syntax

import (
	"sort"

	"gioui.org/op"
)

type Token struct {
	// offset of the start rune in the document.
	Start int
	// offset of the end rune in the document.
	End   int
	Style TokenStyle
}

type TextTokens struct {
	tokens      []Token
	colorScheme *ColorScheme
}

func NewTextTokens(scheme *ColorScheme) *TextTokens {
	return &TextTokens{
		colorScheme: scheme,
	}
}

// Clear the tokens for reuse.
func (t *TextTokens) Clear() {
	t.tokens = t.tokens[:0]
}

func (t *TextTokens) Add(tokenType string, start, end int) {
	style := t.colorScheme.GetTokenStyle(tokenType)
	t.tokens = append(t.tokens, Token{
		Start: start,
		End:   end,
		Style: style,
	})
}

func (t *TextTokens) GetColor(colorID int) op.CallOp {
	cl := t.colorScheme.GetColor(colorID)
	return cl.Op()
}

// Query tokens for rune range. start and end are in runes. start is inclusive
// and end is exclusive. This method assumes the tokens are sorted by start or end
// in ascending order.
func (t *TextTokens) QueryRange(start, end int) []Token {
	if len(t.tokens) == 0 || start >= end {
		return nil
	}

	// Find the index of the first token whose End is greater than start.
	// Tokens before this index cannot overlap because they end too early.
	firstIdx := sort.Search(len(t.tokens), func(i int) bool {
		return t.tokens[i].End > start
	})

	if firstIdx == len(t.tokens) {
		// All tokens end before start, so no overlap.
		return nil
	}

	var result []Token
	for i := firstIdx; i < len(t.tokens); i++ {
		token := t.tokens[i]
		if token.Start < end {
			result = append(result, token)
		} else {
			// This token starts at or after end, no overlap.
			// Since tokens are sorted by Start, we can break early.
			break
		}
	}
	return result
}
