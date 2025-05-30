package syntax

import (
	"fmt"
	"testing"

	"github.com/oligo/gvcode/color"
)

func TestScopeIsValid(t *testing.T) {

	cases := []struct {
		value    string
		expected bool
	}{
		{
			value:    "",
			expected: false,
		},
		{
			value:    "comment",
			expected: true,
		},
		{
			value:    "keyword.control.if",
			expected: true,
		},
		{
			value:    "keyword.control.if.",
			expected: false,
		},
		{
			value:    ".keyword.control.if",
			expected: false,
		},
		{
			value:    "keyword.control..if",
			expected: false,
		},
		{
			value:    ".",
			expected: false,
		},
	}

	for idx, c := range cases {
		t.Run(fmt.Sprintf("case-%d: %s", idx, c.value), func(t *testing.T) {
			scope := StyleScope(c.value)
			if scope.IsValid() != c.expected {
				t.Fail()
			}
		})
	}

}

func TestGetTokenStyle(t *testing.T) {
	scheme := &ColorScheme{}
	scheme.AddStyle("keyword.control", Strikethrough, color.Color{}, color.Color{})

	cases := []struct {
		value    string
		expected bool
	}{
		{
			value:    "keyword.control",
			expected: true,
		},
		{
			value:    "keyword.control.if",
			expected: true,
		},
		{
			value:    "keyword",
			expected: false,
		},
		{
			value:    "keyword.controlx",
			expected: false,
		},
	}

	for idx, c := range cases {
		t.Run(fmt.Sprintf("case-%d: %s", idx, c.value), func(t *testing.T) {
			style := scheme.GetTokenStyle(StyleScope(c.value))
			if (style != 0) != c.expected {
				t.Logf("actual style: %v", style)
				t.Fail()
			}
		})
	}
}
