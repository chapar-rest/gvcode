package decoration

import (
	"gioui.org/op"
)

// Decoration defines APIs each concrete decorations should implement.
// A decoration represents a ranged decoration for a range of text.
type Decoration interface {
	Range() (int, int)
	Source() any
	GetPriority() int
}

// base decoration implements APIs of Decoration.
type baseDecoration struct {
	Src        any
	Priority   int
	Start, End int
}

func (b baseDecoration) Source() any {
	return b.Src
}

func (b baseDecoration) GetPriority() int {
	return b.Priority
}

func (b baseDecoration) Range() (int, int) {
	return b.Start, b.End
}

type Background struct {
	baseDecoration
	// Color for background.
	Color op.CallOp
}

func BackgroundDeco(start, end int, color op.CallOp) Background {
	return Background{
		baseDecoration: baseDecoration{Start: start, End: end},
		Color:          color,
	}
}

type Italic struct {
	baseDecoration
}

func ItalicDeco(start, end int) Italic {
	return Italic{
		baseDecoration: baseDecoration{Start: start, End: end},
	}
}

type Bold struct {
	baseDecoration
}

func BoldDeco(start, end int) Italic {
	return Italic{
		baseDecoration: baseDecoration{Start: start, End: end},
	}
}

type Underline struct {
	baseDecoration
	// Color for the stroke.
	Color op.CallOp
}

func UnderlineDeco(start, end int, color op.CallOp) Underline {
	return Underline{
		baseDecoration: baseDecoration{Start: start, End: end},
		Color:          color,
	}
}

type Squiggle struct {
	baseDecoration
	// Color for the stroke.
	Color op.CallOp
}

func SquiggleDeco(start, end int, color op.CallOp) Squiggle {
	return Squiggle{
		baseDecoration: baseDecoration{Start: start, End: end},
		Color:          color,
	}
}

type Strikethrough struct {
	baseDecoration
	// Color for the stroke.
	Color op.CallOp
}

func StrikethroughDeco(start, end int, color op.CallOp) Strikethrough {
	return Strikethrough{
		baseDecoration: baseDecoration{Start: start, End: end},
		Color:          color,
	}
}

type Box struct {
	baseDecoration
	// Color for the stroke.
	Color op.CallOp
}

func BoxDeco(start, end int, color op.CallOp) Box {
	return Box{
		baseDecoration: baseDecoration{Start: start, End: end},
		Color:          color,
	}
}

// type IndentGuide struct {
// 	baseDecoration
// 	// Color for the stroke.
// 	Color op.CallOp
// 	// Width is the line width.
// 	Width unit.Dp
// }

// func (d *IndentGuide) Kind() DecoKind {
// 	return IndentGuideKind
// }

// type InlayText struct {
// 	baseDecoration
// 	// Color for text.
// 	Color op.CallOp
// 	// Text for InlayText kind
// 	Text string
// }

// func (d InlayText) Kind() DecoKind {
// 	return InlayTextKind
// }
