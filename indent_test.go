package gvcode

import (
	"fmt"
	"testing"

	"gioui.org/layout"
	"gioui.org/text"
	"gioui.org/unit"
	"github.com/oligo/gvcode/internal/buffer"
)

func TestDedentLine(t *testing.T) {
	text := &textView{}
	text.SetSource(buffer.NewTextSource())
	text.TabWidth = 4

	cases := []struct {
		input string
		want  string
	}{
		{
			input: "abc",
			want:  "abc",
		},
		{
			input: "\t\tabc",
			want:  "\tabc",
		},

		{
			input: "\t    abc",
			want:  "\tabc",
		},

		{
			input: "\t      abc",
			want:  "\t    abc",
		},
		{
			input: "    abc",
			want:  "abc",
		},
		{
			input: "   abc",
			want:  "abc",
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d: %s", i, tc.input), func(t *testing.T) {
			actual := text.dedentLine(tc.input)
			if actual != tc.want {
				t.Fail()
			}
		})
	}

}

func TestIndentOnBreak(t *testing.T) {
	setup := func(input string, initialPos int) *textView {
		vw := &textView{TabWidth: 4, SoftTab: false, TextSize: unit.Sp(14)}
		vw.SetSource(buffer.NewTextSource())
		vw.SetText(input)

		gtx := layout.Context{}
		shaper := text.NewShaper()
		vw.Layout(gtx, shaper)

		vw.SetCaret(initialPos, initialPos)
		return vw
	}

	cases := []struct {
		input      string
		initialPos int
		want       string
		wantMoves  int
	}{
		{
			input:      "abc",
			initialPos: 3,
			want:       "abc\n",
			wantMoves:  1,
		},
		{
			input:      "\tabcde",
			initialPos: 4,
			want:       "\tabc\n\tde",
			wantMoves:  2,
		},
		{
			input:      "abc{\n}",
			initialPos: 4,
			want:       "abc{\n\t\n}",
			wantMoves:  2,
		},

		{
			input:      "abc{de\n}",
			initialPos: 6,
			want:       "abc{de\n\t\n}",
			wantMoves:  2,
		},
		{
			input:      "abc{}",
			initialPos: 4,
			want:       "abc{\n\t\n}",
			wantMoves:  3,
		},
		{
			input:      "\tabc{\n\n}",
			initialPos: 6,
			want:       "\tabc{\n\n\n}",
			wantMoves:  1,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d: %s", i, tc.input), func(t *testing.T) {
			text := setup(tc.input, tc.initialPos)
			actual := text.IndentOnBreak("\n")
			finalContent := text.src.Text(nil)
			if actual != tc.wantMoves || string(finalContent) != tc.want {
				t.Logf("want content: %q, actual content: %q, want moves: %d, actual moves: %d", tc.want, string(finalContent), tc.wantMoves, actual)
				t.Fail()
			}
		})
	}

}
