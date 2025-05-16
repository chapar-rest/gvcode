package syntax

import (
	"fmt"
	"image"
	"image/color"
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

	doc := "Hello,world"

	scheme := &ColorScheme{}
	scheme.AddTokenType("t1", Bold|Underline, color.NRGBA{R: 200}, color.NRGBA{G: 200})
	scheme.AddTokenType("t2", Bold, color.NRGBA{R: 200}, color.NRGBA{G: 200})
	line := layoutText(doc)



	testcases := []struct {
		tokens   []Token
		wantSize int   // the number of runs expected.
		wantLen  []int // the number of glyphs(or runes for simple character) expected.
	}{
		// case1: no tokens
		{
			tokens:   []Token{},
			wantSize: 1,
			wantLen:  []int{11},
		},
		// unstyled text between tokens.
		{
			tokens:   []Token{{TokenType: "t1", Start: 0, End: 5}, {TokenType: "t1", Start: 6, End: 11}},
			wantSize: 3,
			wantLen:  []int{5, 1, 5},
		},
		// continuous tokens with no gapped text.
		{
			tokens:   []Token{{TokenType: "t1", Start: 0, End: 5}, {TokenType: "t1", Start: 5, End: 11}},
			wantSize: 2,
			wantLen:  []int{5, 6},
		},
	}

	for i, tc := range testcases {
		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
			tokens := NewTextTokens(scheme)
			tokens.Set(tc.tokens...)

			var runs []painter.RenderRun
			tokens.Split(line, &runs)
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
