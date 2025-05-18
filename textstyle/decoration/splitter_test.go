package decoration

import (
	"fmt"
	"image"
	"testing"

	"gioui.org/layout"
	"gioui.org/text"
	"github.com/oligo/gvcode/internal/buffer"
	lt "github.com/oligo/gvcode/internal/layout"
	"github.com/oligo/gvcode/internal/painter"
	"golang.org/x/image/math/fixed"
)

func TestLineSplit(t *testing.T) {
	layoutText := func(doc string) *lt.Line {
		gtx := layout.Context{Constraints: layout.Constraints{Max: image.Point{X: 1e6, Y: 1e6}}}

		buf := buffer.NewTextSource()
		buf.SetText([]byte(doc))
		layouter := lt.NewTextLayout(buf)
		textSize := fixed.I(gtx.Sp(14))
		layouter.Layout(text.NewShaper(), &text.Parameters{PxPerEm: textSize}, 4, false)

		return layouter.Lines[0]
	}

	doc := "Hello,world\nanother line"

	line := layoutText(doc)

	testcases := []struct {
		decos    []Decoration
		wantSize int   // the number of runs expected.
		wantLen  []int // the number of glyphs(or runes for simple character) expected.
	}{
		// case1: no decorations
		{
			decos:    nil,
			wantSize: 0,
			wantLen:  []int{},
		},
		// unstyled text between decorations.
		{
			decos:    []Decoration{{Source: "t1", Start: 0, End: 5}, {Source: "t1", Start: 6, End: 11}},
			wantSize: 2,
			wantLen:  []int{5, 5},
		},
		// continuous decos with no gapped text.
		{
			decos:    []Decoration{{Source: "t1", Start: 0, End: 5}, {Source: "t1", Start: 5, End: 11}},
			wantSize: 2,
			wantLen:  []int{5, 6},
		},
		// unstyled leading and trailing text.
		{
			decos:    []Decoration{{Source: "t1", Start: 2, End: 5}},
			wantSize: 1,
			wantLen:  []int{3},
		},
		// overlapped decorations.
		{
			decos:    []Decoration{{Source: "t1", Start: 2, End: 6}, {Source: "t1", Start: 3, End: 11}},
			wantSize: 2,
			wantLen:  []int{4, 8},
		},

		// decorations across multiple lines.
		// {
		// 	decos:    []Decoration{{Source: "t1", Start: 2, End: 6}, {Source: "t1", Start: 10, End: 19}},
		// 	wantSize: 2,
		// 	wantLen:  []int{4, 8},
		// },
	}

	for i, tc := range testcases {
		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
			tree := NewDecorationTree()
			tree.Insert(tc.decos...)
			var runs []painter.RenderRun
			tree.Split(line, &runs)
			if len(runs) != tc.wantSize {
				t.FailNow()
			}

			ii := 0
			for _, r := range runs {
				want := tc.wantLen[ii]
				if want != len(r.Glyphs) {
					t.Fail()
				}
				ii++
			}
		})
	}
}
