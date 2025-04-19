package gvcode

import (
	"fmt"
	"testing"

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
