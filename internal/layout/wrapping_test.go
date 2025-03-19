package layout

import (
	"fmt"
	"log"
	"math"
	"testing"

	"gioui.org/font"
	"gioui.org/text"
	"golang.org/x/image/math/fixed"
)

func TestWrapParagraph(t *testing.T) {
	shaper := text.NewShaper()

	params := text.Parameters{
		Font:     font.Font{Typeface: font.Typeface("monospace")},
		PxPerEm:  fixed.I(14),
		MaxWidth: 1e6,
	}

	shaper.LayoutString(params, "\u0020")
	spaceGlyph, _ := shaper.NextGlyph()

	calculateWidth := func(input string) int {
		// works only for single rune grapheme clusters.
		width := spaceGlyph.Advance.Mul(fixed.I(len([]rune(input)))).Ceil()
		log.Println("width: ", width)
		return width
	}

	testcases := []struct {
		input string
	}{
		{
			input: "alonglongword",
		},
		{
			input: "word\n",
		},
	}

	for i, tc := range testcases {
		t.Run(fmt.Sprintf("%d: %s", i, tc.input), func(t *testing.T) {
			width := calculateWidth(tc.input)
			shaper := text.NewShaper()
			shaper.LayoutString(params, tc.input)

			lineWidth := int(math.Ceil(float64(width) / 2.0))

			wrapper := lineWrapper{}
			lines := wrapper.WrapParagraph(glyphIter{shaper: shaper}.All(), []rune(tc.input), lineWidth, 4, &spaceGlyph)

			runes := 0
			for _, line := range lines {
				runes += line.Runes
			}

			if len(lines) < 2 || runes != len([]rune(tc.input)) {
				t.Fail()
			}
		})
	}
}
