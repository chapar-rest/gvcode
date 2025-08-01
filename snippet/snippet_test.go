package snippet

import "testing"

func TestSnippetParse(t *testing.T) {
	snippet := `for (const ${2:element} of ${1:array}) {", "\t$0", $TM_CURRENT_LINE"}`
	snp := NewSnippet(snippet)
	err := snp.Parse()
	if err != nil {
		t.FailNow()
	}

	t.Logf("template: %s", snp.Template())
	for idx, ts := range snp.TabStops() {
		start, end := snp.TabStopOff(idx)
		t.Logf("tabstop: %v, off: [%d-%d]", ts, start, end)
	}

	t.Fail()
}
