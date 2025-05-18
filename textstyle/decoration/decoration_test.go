package decoration

import (
	"testing"
)

func TestInsertDecoration(t *testing.T) {
	d := NewDecorationTree()

	bg := Decoration{Start: 0, End: 5, Background: &Background{}}
	italic := Decoration{Start: 6, End: 9, Italic: true}
	bold := Decoration{Start: 6, End: 9, Bold: true}
	underline := Decoration{Start: 0, End: 6, Underline: &Underline{}}
	strikethrough := Decoration{Start: 11, End: 15, Strikethrough: &Strikethrough{}}
	box := Decoration{Start: 16, End: 20, Border: &Border{}}

	d.Insert(bg)
	d.Insert(italic)
	d.Insert(bold)
	d.Insert(underline)
	d.Insert(strikethrough)
	d.Insert(box)

	all := d.QueryRange(0, 20)
	if len(all) != 6 {
		t.Fail()
	}
}

func TestRemoveDecorationBySource(t *testing.T) {
	d := NewDecorationTree()

	bg := Decoration{Start: 0, End: 5, Background: &Background{}}
	italic := Decoration{Start: 6, End: 9, Italic: true}
	bold := Decoration{Start: 6, End: 9, Bold: true}
	underline := Decoration{Start: 0, End: 6, Underline: &Underline{}}
	strikethrough := Decoration{Start: 11, End: 15, Strikethrough: &Strikethrough{}}
	box := Decoration{Start: 16, End: 20, Border: &Border{}}
	bg.Source = "selection"
	box.Source = "selection"

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
}
