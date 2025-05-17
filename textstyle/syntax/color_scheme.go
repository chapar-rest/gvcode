package syntax

import (
	"slices"

	"github.com/oligo/gvcode/color"
)

// ColorScheme defines the token types and their styles used for syntax highlighting.
type ColorScheme struct {
	// Name is the name of the color scheme.
	Name string
	// Foreground provides a default text color for the editor.
	Foreground color.Color
	// Background provides a default text color for the editor.
	Background color.Color

	color.ColorPalette
	// tokenTypes are registered token types for the color scheme.
	// It can be mapped to captures for Tree-Sitter, and TokenType of Chroma.
	tokenTypes []string

	// styles maps tokenType index to non-packed token style.
	styles map[int]*tokenStyleRaw
}

type tokenStyleRaw struct {
	textStyle TextStyle
	fg, bg    int
}

func (cs *ColorScheme) addTokenType(tokenType string) int {
	if idx := slices.Index(cs.tokenTypes, tokenType); idx >= 0 {
		return idx
	}

	cs.tokenTypes = append(cs.tokenTypes, tokenType)
	return len(cs.tokenTypes) - 1
}

func (cs *ColorScheme) getTokenStyle(id int) *tokenStyleRaw {
	if style, exists := cs.styles[id]; exists {
		return style
	} else {
		return nil
	}
}

func (cs *ColorScheme) AddTokenType(tokenType string, textStyle TextStyle, fg, bg color.Color) {
	tokenTypeID := cs.addTokenType(tokenType)
	fgID := cs.AddColor(fg)
	bgID := cs.AddColor(bg)

	if cs.styles == nil {
		cs.styles = make(map[int]*tokenStyleRaw)
	}

	cs.styles[tokenTypeID] = &tokenStyleRaw{
		textStyle: textStyle,
		fg:        fgID,
		bg:        bgID,
	}
}

func (cs *ColorScheme) GetTokenStyleByID(tokenTypeID int) StyleMeta {
	style := cs.getTokenStyle(tokenTypeID)
	if style == nil {
		return StyleMeta(0)
	}

	return packTokenStyle(tokenTypeID, style.fg, style.bg, style.textStyle)
}

func (cs *ColorScheme) GetTokenStyle(tokenType string) StyleMeta {
	idx := slices.Index(cs.tokenTypes, tokenType)
	if idx < 0 {
		return StyleMeta(0)
	}

	style := cs.getTokenStyle(idx)
	if style == nil {
		return StyleMeta(0)
	}

	return packTokenStyle(idx, style.fg, style.bg, style.textStyle)
}
