package painter

import (
	"fmt"
	"image"
	"image/color"
	"testing"

	"gioui.org/layout"
	"gioui.org/text"
	"github.com/oligo/gvcode/internal/buffer"
	lt "github.com/oligo/gvcode/internal/layout"
	"github.com/oligo/gvcode/textstyle/syntax"
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

	scheme := &syntax.ColorScheme{}
	scheme.AddTokenType("t1", syntax.Bold, color.NRGBA{R: 200}, color.NRGBA{G: 200})
	scheme.AddTokenType("t2", syntax.Bold, color.NRGBA{R: 200}, color.NRGBA{G: 200})
	line := layoutText(doc)

	type token struct {
		tokenType  string
		start, end int
	}

	testcases := []struct {
		tokens     []token
		wantSize   int // the number of runs expected.
		wantRanges [][]int
	}{
		// case1: no tokens
		{
			tokens:   []token{},
			wantSize: 1,
			wantRanges: [][]int{
				{0, 11},
			},
		},
		// unstyled text between tokens.
		{
			tokens:   []token{{tokenType: "t1", start: 0, end: 5}, {tokenType: "t1", start: 6, end: 11}},
			wantSize: 3,
			wantRanges: [][]int{
				{0, 5}, {5, 6}, {6, 11},
			},
		},
		// continuous tokens with no gapped text.
		{
			tokens:   []token{{tokenType: "t1", start: 0, end: 5}, {tokenType: "t1", start: 5, end: 11}},
			wantSize: 2,
			wantRanges: [][]int{
				{0, 5}, {5, 11},
			},
		},
	}

	splitter := lineSplitter{}
	for i, tc := range testcases {
		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
			tokens := syntax.NewTextTokens(scheme)
			for _, tk := range tc.tokens {
				tokens.Add(tk.tokenType, tk.start, tk.end)
			}

			splitter.Split(line, tokens)
			if splitter.Size() != tc.wantSize {
				t.FailNow()
			}

			ii := 0
			for r := range splitter.Runs() {
				want := tc.wantRanges[ii]
				if want[0] != r.start || want[1] != r.end {
					t.Fail()
				}
				ii++
			}
		})
	}

}
