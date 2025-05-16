package decoration

import (
	"cmp"
	"errors"

	"github.com/rdleal/intervalst/interval"
)

// DecorationTree leverages a interval tree to stores overlapping decorations.
type DecorationTree struct {
	tree *interval.MultiValueSearchTree[Decoration, int]
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
func (d *DecorationTree) Insert(deco Decoration) {
	start, end := deco.Range()
	d.tree.Insert(start, end, deco)
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
	// TODO
	maxVals, found := d.tree.MaxEnd()
	if !found {
		return errors.New("no decoration found")
	}

	_, end := maxVals[0].Range()
	all, found := d.tree.AllIntersections(0, end)
	if !found {
		return errors.New("no decoration found")
	}

	for _, deco := range all {
		if deco.Source() == source {
			s, e := deco.Range()
			d.tree.Delete(s, e)
		}
	}

	return nil
}

// func (d *DecorationTree) All() iter.Seq[RangeDecoration] {
// 	return func(yield func(RangeDecoration) bool) {
// 		for _, k := range d.segments {
// 			if !yield(k) {
// 				return
// 			}
// 		}

// 		d.intervals.
// 	}
// }
