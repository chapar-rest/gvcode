package gvcode

import (
	"maps"

	"gioui.org/io/key"
)

// Bracket pairs
var ltrBracketPairs = map[string]string{
	"(": ")",
	"{": "}",
	"[": "]",
}
var rtlBracketPairs = map[string]string{
	"(": ")",
	"{": "}",
	"[": "]",
}
var quotePairs = map[string]string{
	"'":  "'",
	"\"": "\"",
	"`":  "`",
}

var autoCompletablePairs = mergeMaps(ltrBracketPairs, quotePairs)

func (e *Editor) autoCompleteTextPair(ke key.EditEvent) bool {
	closing, ok := autoCompletablePairs[ke.Text]
	if !ok {
		return false
	}

	e.scrollCaret = true
	e.scroller.Stop()
	e.replace(ke.Range.Start, ke.Range.End, ke.Text+closing)
	e.text.MoveCaret(-len([]rune(closing)), -len([]rune(closing)))
	return true
}

func mergeMaps(sources ...map[string]string) map[string]string {
	dest := make(map[string]string)
	for _, src := range sources {
		maps.Copy(dest, src)
	}

	return dest
}
