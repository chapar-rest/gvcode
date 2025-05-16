package gvcode

import (
	"fmt"
	"testing"

	"gioui.org/layout"
	"gioui.org/text"
	"github.com/oligo/gvcode/internal/buffer"
)

func TestNearestMatchingBrackets(t *testing.T) {
	view := &textView{}
	view.SetSource(buffer.NewTextSource())
	gtx := layout.Context{}
	shaper := text.NewShaper()

	setup := func(input string, caret int) func() {
		return func() {
			view.SetText(input)
			view.Layout(gtx, shaper)
			view.SetCaret(caret, caret)
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
			want:  []int{0, 6},
		},

		{
			setup: setup("{a[b]c}", 2),
			want:  []int{2, 4},
		},
		{
			setup: setup("{a[b]c}", 3),
			want:  []int{2, 4},
		},
		{
			setup: setup("{a[b]cde}", 6),
			want:  []int{0, 8},
		},
		{
			setup: setup("{ab)c}", 3),
			want:  []int{-1, 3},
		},
		{
			setup: setup("{ab(c}", 3),
			want:  []int{3, -1},
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("case %d", i), func(t *testing.T) {
			tc.setup()
			left, right := view.NearestMatchingBrackets()
			if left != tc.want[0] || right != tc.want[1] {
				t.Fail()
			}
		})
	}
}
