package gvcode

import (
	"fmt"
	"testing"

	"gioui.org/layout"
	"gioui.org/text"
	"github.com/oligo/gvcode/internal/buffer"
)

func TestNearestMatchingBrackets(t *testing.T) {
	bh := bracketHandler{textView: &textView{}}
	bh.textView.SetSource(buffer.NewTextSource())
	gtx := layout.Context{}
	shaper := text.NewShaper()

	setup := func(input string, caret int) func() {
		return func() {
			bh.SetText(input)
			bh.Layout(gtx, shaper)
			bh.SetCaret(caret, caret)
		}
	}

	cases := []struct {
		setup func()
		want  []int
	}{
		{
			setup: setup("{abc}", 0),
			want:  []int{0, 4},
		},

		{
			setup: setup("{abc}", 4),
			want:  []int{0, 4},
		},

		{
			setup: setup("{abc}", 1),
			want:  []int{0, 4},
		},

		{
			setup: setup("{a[b]c}", 0),
			want: []int{0, 6},
		},

		{
			setup: setup("{a[b]c}", 2),
			want: []int{2, 4},
		},
		{
			setup: setup("{a[b]c}", 3),
			want: []int{2, 4},
		},
		{
			setup: setup("{a[b]cde}", 6),
			want: []int{0, 8},
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("case %d", i), func(t *testing.T) {
			tc.setup()
			left, right := bh.NearestMatchingBrackets()
			if left != tc.want[0] || right != tc.want[1] {
				t.Fail()
			}
		})
	}
}
