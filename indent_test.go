package gvcode

import (
	"fmt"
	"testing"
)

func TestDedentLine(t *testing.T) {
	indenter := autoIndenter{Editor: &Editor{}}
	indenter.Editor.WithOptions(WithTabWidth(4))

	cases := []struct{
		input string
		want string
	}{
		{
			input: "abc",
			want: "abc",
		},
		{
			input: "\t\tabc",
			want: "\tabc",
		},

		{
			input: "\t    abc",
			want: "\tabc",
		},

		{
			input: "\t      abc",
			want: "\t    abc",
		},
		{
			input: "    abc",
			want: "abc",
		},
		{
			input: "   abc",
			want: "abc",
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d: %s", i, tc.input,), func(t *testing.T) {
			actual := indenter.dedentLine(tc.input)
			if actual != tc.want {
				t.Fail()
			}
		})
	} 

}
