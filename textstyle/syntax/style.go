package syntax

import (
	"fmt"
)

type TextStyle uint8

const (
	Unset TextStyle = 1 << iota
	Bold
	Italic
	Underline
	Strikethrough
	Border
)

func (s TextStyle) HasStyle(mask TextStyle) bool {
	return s&mask > 0
}

const (
	textStyleOffset  = 0
	backgroundOffset = 5
	foregroundOffset = 13
	tokenTypeOffset  = 21
	languageIdOffset = 28

	languageIdMask = 0b11110000_00000000_00000000_00000000
	tokenTypeMask  = 0b00001111_11100000_00000000_00000000
	foregroundMask = 0b00000000_00011111_11100000_00000000
	backgroundMask = 0b00000000_00000000_00011111_11100000
	textStyleMask  = 0b00000000_00000000_00000000_00011111
)

// TokenStyles applies a bit packed binary format to encode tokens from
// syntax parser, to be used as styles for text rendering. This is like
// TokenMetadata in Monaco/vscode.
//
// It uses 4 bytes to hold the metadata, and the layout is as follows:
// Bits:  31   ...   0
// [4][7][9][8][4] = 32
// |  |  |  |  |
// |  |  |  |  └── Text style flags (5bits, bold, italic, underline, strikethrough, border)
// |  |  |  └───── Background color ID (8bits, 0–255)
// |  |  └──────── Foreground color ID (8bits, 0–255)
// |  └─────────── Token type (7bits, 0–127)
// └────────────── Language ID (4bits, 0–15)
//
// The color IDs are mapped to indices of color palette.
type TokenStyle uint32

func (t TokenStyle) LanguageID() int {
	return int((t & languageIdMask) >> languageIdOffset)
}

func (t TokenStyle) TokenType() int {
	return int(t & tokenTypeMask >> tokenTypeOffset)
}

func (t TokenStyle) Foreground() int {
	return int(t & foregroundMask >> foregroundOffset)
}

func (t TokenStyle) Background() int {
	return int(t & backgroundMask >> backgroundOffset)
}

func (t TokenStyle) FontStyle() TextStyle {
	return TextStyle(t & textStyleMask >> textStyleOffset)
}

func (t TokenStyle) String() string {
	return fmt.Sprintf("Lang=%d Type=%d FG=%d BG=%d Style=%04b",
		t.LanguageID(), t.TokenType(), t.Foreground(), t.Background(), t.FontStyle())
}

func PackTokenStyle(langID int, tokenType int, fg, bg int, textStyles TextStyle) TokenStyle {
	s := TokenStyle(0)

	s ^= TokenStyle(langID << languageIdOffset)
	s ^= TokenStyle(tokenType << tokenTypeOffset)
	s ^= TokenStyle(fg << foregroundOffset)
	s ^= TokenStyle(bg << backgroundOffset)
	s ^= TokenStyle(textStyles << textStyleOffset)
	return s
}
