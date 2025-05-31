package textview

import (
	"fmt"
	"testing"

	"gioui.org/layout"
	"gioui.org/text"
	"github.com/oligo/gvcode/internal/buffer"
)

func TestReadWord(t *testing.T) {
	view := &TextView{}
	view.SetSource(buffer.NewTextSource())

	doc := "hello,world!!!"

	testcases := []struct {
		position int
		want     struct {
			word   string
			offset int
		}
	}{
		{
			position: 0,
			want: struct {
				word   string
				offset int
			}{word: "hello", offset: 0},
		},
		{
			position: 2,
			want: struct {
				word   string
				offset int
			}{word: "hello", offset: 2},
		},

		{
			position: 5,
			want: struct {
				word   string
				offset int
			}{word: "hello", offset: 5},
		},

		{
			position: 6,
			want: struct {
				word   string
				offset int
			}{word: "world", offset: 0},
		},

		{
			position: 11,
			want: struct {
				word   string
				offset int
			}{word: "world", offset: 5},
		},
		{
			position: 12,
			want: struct {
				word   string
				offset int
			}{word: "", offset: 0},
		},
	}

	for i, tc := range testcases {
		t.Run(fmt.Sprintf("case %d", i), func(t *testing.T) {
			view.SetText(doc)
			gtx := layout.Context{}
			shaper := text.NewShaper()
			view.Layout(gtx, shaper)
			view.SetCaret(tc.position, tc.position)
			w, o := view.ReadWord(false)
			if w != tc.want.word || o != tc.want.offset {
				t.Logf("want: [word: %s, offset: %d], actual: [word: %s, offset: %d]", tc.want.word, tc.want.offset, w, o)
				t.Fail()
			}
		})
	}

}
