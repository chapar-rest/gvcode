package gvcode

import (
	"log/slog"

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
	e.initBuffer()
	e.text.decorations.Insert(styles...)
}

func (e *Editor) ClearDecorations(source string) {
	e.initBuffer()

	if source == "" {
		e.text.decorations.RemoveAll()
	} else {
		e.text.decorations.RemoveBySource(source)
	}
}

func (e *Editor) SetSyntaxTokens(tokens ...syntax.Token) {
	e.initBuffer()
	if e.colorScheme == nil {
		slog.Info("No color scheme configured.")
		return
	}
	e.text.syntaxStyles.Set(tokens...)
}
