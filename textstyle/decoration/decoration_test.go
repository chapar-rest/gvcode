package decoration

import (
	"testing"

	"gioui.org/op"
)

func TestInsertDecoration(t *testing.T) {
	d := NewDecorationTree()

	color := op.CallOp{}

	bg := BackgroundDeco(0, 5, color)
	italic := ItalicDeco(6, 9)
	bold := BoldDeco(7, 10)
	underline := UnderlineDeco(0, 6, color)
	strikethrough := StrikethroughDeco(11, 15, color)
	box := BoxDeco(16, 20, color)

	d.Insert(bg)
	d.Insert(italic)
	d.Insert(bold)
	d.Insert(underline)
	d.Insert(strikethrough)
	d.Insert(box)

	//t.Fail()
	all := d.QueryRange(0, 20)
	if len(all) != 6 {
		t.Fail()
	}
}

func TestRemoveDecorationBySource(t *testing.T) {
	d := NewDecorationTree()

	color := op.CallOp{}

	bg := BackgroundDeco(0, 5, color)
	bg.Src = "selection"
	italic := ItalicDeco(6, 9)
	bold := BoldDeco(7, 10)
	underline := UnderlineDeco(0, 6, color)
	strikethrough := StrikethroughDeco(11, 15, color)
	box := BoxDeco(16, 20, color)
	box.Src = "selection"

	d.Insert(bg)
	d.Insert(italic)
	d.Insert(bold)
	d.Insert(underline)
	d.Insert(strikethrough)
	d.Insert(box)

	d.RemoveBySource("selection")
	if v := d.QueryRange(0, 5); len(v) != 1 {
		t.Fail()
	}

	if v := d.QueryRange(16, 20); len(v) > 0 {
		t.Fail()
	}

	// for _, seg := range d.QueryRange(0, 20) {
	// 	start, end := seg.Range()
	// 	t.Logf("range: (%d, %d), styles: %d", start, end, seg.Source())
	// }
}
