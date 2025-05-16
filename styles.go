package gvcode

import (
	"github.com/oligo/gvcode/textstyle/decoration"
	"github.com/oligo/gvcode/textstyle/syntax"
)

// TextRange contains the range of text of interest in the document. It can used for
// search, styling text, or any other purposes.
type TextRange struct {
	// offset of the start rune in the document.
	Start int
	// offset of the end rune in the document.
	End int
}

func (e *Editor) AddDecorations(styles ...decoration.Decoration) {
	if e.decorations == nil {
		e.decorations = decoration.NewDecorationTree()
	}

	for _, style := range styles {
		e.decorations.Insert(style)
	}
}

func (e *Editor) SetSyntaxTokens(tokens ...syntax.Token) {
	e.syntaxStyles.Set(tokens...)
}
