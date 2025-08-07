package decoration

import (
	"cmp"
	"errors"

	"github.com/oligo/gvcode/color"
	"github.com/oligo/gvcode/internal/buffer"
	"github.com/oligo/gvcode/internal/layout"
	"github.com/oligo/gvcode/internal/painter"
	"github.com/rdleal/intervalst/interval"
)

type Background struct {
	// Color for background.
	Color color.Color
}

type Underline struct {
	// Color for the stroke.
	Color color.Color
}

type Squiggle struct {
	// Color for the stroke.
	Color color.Color
}

type Strikethrough struct {
	// Color for the stroke.
	Color color.Color
}

type Border struct {
	// Color for the stroke.
	Color color.Color
}

// Decoration defines APIs each concrete decorations should implement.
// A decoration represents styles sharing between a range of text.
type Decoration struct {
	// Source marks where the decoration is from.
	Source any
	// Priority configures the painting order of the decoration.
	Priority int
	// Start and End are rune offset in the document.
	Start, End    int
	Background    *Background
	Underline     *Underline
	Squiggle      *Squiggle
	Strikethrough *Strikethrough
	Border        *Border
	Italic        bool
	Bold          bool
	startMarker   *buffer.Marker
	endMarker     *buffer.Marker
}

// Bind binds the decoration the the text source. This adds the start
// and end position as markers to the text source.
func (d *Decoration) Bind(src buffer.TextSource) error {
	if d.Start < 0 || d.End < 0 {
		return errors.New("invalid decoration range")
	}

	markerStart, err := src.CreateMarker(d.Start, buffer.BiasBackward)
	if err != nil {
		return err
	}
	markerEnd, err := src.CreateMarker(d.End, buffer.BiasForward)
	if err != nil {
		return err
	}
	d.startMarker = markerStart
	d.endMarker = markerEnd
	return nil
}

func (d *Decoration) Range() (start, end *buffer.Marker) {
	return d.startMarker, d.endMarker
}

// func (d *Decoration) CheckValid() error {
// 	if d.Source == nil {
// 		return errors.New("missing source")
// 	}
// 	if d.Start < 0 || d.End < 0 || d.Start >= d.End {
// 		return errors.New("invalid decoration range")
// 	}
// 	if d.Background != nil && !d.Background.Color.IsSet() {
// 		return errors.New("invalid background")
// 	}
// 	if d.Underline != nil && !d.Underline.
// }

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

// DecorationTree leverages a interval tree to stores overlapping decorations.
type DecorationTree struct {
	tree         *interval.MultiValueSearchTree[Decoration, int]
	lineSplitter decorationLineSplitter
}

func NewDecorationTree() *DecorationTree {
	tree := interval.NewMultiValueSearchTree[Decoration](func(a, b int) int {
		return cmp.Compare(a, b)
	})

	return &DecorationTree{
		tree: tree,
	}
}

// Insert a new decoration range. start and end are offset in rune in the document.
func (d *DecorationTree) Insert(decos ...Decoration) {
	for _, deco := range decos {
		d.tree.Insert(deco.Start, deco.End, deco)
	}
}

// Query returns all styles at a given character offset
func (d *DecorationTree) Query(pos int) []Decoration {
	all, _ := d.tree.AllIntersections(pos, pos+1)
	return all
}

// QueryRange returns all segments overlapping the range
func (d *DecorationTree) QueryRange(start, end int) []Decoration {
	if start >= end {
		return nil
	}

	all, _ := d.tree.AllIntersections(start, end)
	return all
}

func (d *DecorationTree) RemoveBySource(source string) error {
	maxVals, found := d.tree.MaxEnd()
	if !found {
		return errors.New("no decoration found")
	}

	end := maxVals[0].End
	all, found := d.tree.AllIntersections(0, end)
	if !found {
		return errors.New("no decoration found")
	}

	for _, deco := range all {
		if deco.Source == source {
			d.tree.Delete(deco.Start, deco.End)
		}
	}

	return nil
}

func (d *DecorationTree) RemoveAll() {
	maxVals, found := d.tree.MaxEnd()
	if !found {
		return
	}

	end := maxVals[0].End
	all, found := d.tree.AllIntersections(0, end)
	if !found {
		return
	}

	for _, deco := range all {
		d.tree.Delete(deco.Start, deco.End)
	}

}

// Refresh checks if any decoration range has changed and rebuilds the interval
// tree if necessary.
func (d *DecorationTree) Refresh() {
	maxVals, found := d.tree.MaxEnd()
	if !found {
		return
	}

	end := maxVals[0].End
	all, found := d.tree.AllIntersections(0, end)
	if !found {
		return
	}

	invalid := false

	for i := range all {
		deco := all[i]
		start, end := deco.Range()
		if start.Offset() != deco.Start || end.Offset() != deco.End {
			//log.Printf("old: %d-%d, new: %d-%d", deco.Start, deco.End, start.Offset(), end.Offset())
			invalid = true
			all[i].Start = start.Offset()
			all[i].End = end.Offset()
		}
	}

	if invalid {
		d.tree = interval.NewMultiValueSearchTree[Decoration](func(a, b int) int {
			return cmp.Compare(a, b)
		})
		d.Insert(all...)
	}
}

// Split implements painter.LineSplitter
func (t *DecorationTree) Split(line *layout.Line, runs *[]painter.RenderRun) {
	t.lineSplitter.Split(line, t, runs)
}
