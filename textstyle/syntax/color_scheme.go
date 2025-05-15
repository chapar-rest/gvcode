package syntax

import (
	"image/color"
	"slices"

	"gioui.org/op"
	"gioui.org/op/paint"
)

type Color struct {
	val color.NRGBA
	op  op.CallOp
}

func (c *Color) NRGBA() color.NRGBA {
	return c.val
}

func (c *Color) makeOp() {
	if c.op != (op.CallOp{}) {
		return
	}
	ops := new(op.Ops)
	m := op.Record(ops)
	paint.ColorOp{Color: c.val}.Add(ops)
	c.op = m.Stop()
}

func (c *Color) Op() op.CallOp {
	if c.val == (color.NRGBA{}) {
		return op.CallOp{}
	}

	c.makeOp()
	return c.op
}

// ColorScheme defines the token types and their styles used for syntax highlighting.
type ColorScheme struct {
	// Name is the name of the color scheme.
	Name string
	// colors holds the color palette of the color scheme.
	colors []*Color
	// tokenTypes are registered token types for the color scheme.
	// It can be mapped to captures for Tree-Sitter, and TokenType of Chroma.
	tokenTypes []string

	// styles maps tokenType index to non-packed token style.
	styles map[int]*tokenStyleRaw
}

type tokenStyleRaw struct {
	textStyle  TextStyle
	fg, bg int
}

func (cs *ColorScheme) GetColor(id int) Color {
	if id < 0 || id >= len(cs.colors) {
		return Color{}
	}

	c := cs.colors[id]
	c.makeOp()
	return *c
}

func (cs *ColorScheme) addColor(cl color.NRGBA) int {
	if idx := slices.IndexFunc(cs.colors, func(c *Color) bool { return c.val == cl }); idx >= 0 {
		return idx
	}

	cs.colors = append(cs.colors, &Color{val: cl})
	return len(cs.colors) - 1
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

func (cs *ColorScheme) AddTokenType(tokenType string, textStyle TextStyle, fg, bg color.NRGBA) {
	tokenTypeID := cs.addTokenType(tokenType)
	fgID := cs.addColor(fg)
	bgID := cs.addColor(bg)

	if cs.styles == nil {
		cs.styles = make(map[int]*tokenStyleRaw)
	}

	cs.styles[tokenTypeID] = &tokenStyleRaw{
		textStyle: textStyle,
		fg:        fgID,
		bg:        bgID,
	}
}

func (cs *ColorScheme) GetTokenStyleByID(tokenTypeID int) TokenStyle {
	style := cs.getTokenStyle(tokenTypeID)
	if style == nil {
		return TokenStyle(0)
	}

	return PackTokenStyle(0, tokenTypeID, style.fg, style.bg, style.textStyle)
}

func (cs *ColorScheme) GetTokenStyle(tokenType string) TokenStyle {
	idx := slices.Index(cs.tokenTypes, tokenType)
	if idx < 0 {
		return TokenStyle(0)
	}

	style := cs.getTokenStyle(idx)
	if style == nil {
		return TokenStyle(0)
	}

	return PackTokenStyle(0, idx, style.fg, style.bg, style.textStyle)
}
