package snippet

import "testing"

func TestSnippetParse(t *testing.T) {
	snippet := `for (const ${2:element} of ${1:array}) {", "\t$0", $TM_CURRENT_LINE"}`
	snp := NewSnippet(snippet)
	err := snp.Parse()
	if err != nil {
		t.FailNow()
	}

	expectedTemplate := `for (const element of array) {", "\t", "}`
	if snp.Template() != expectedTemplate {
		t.Logf("template: %s", snp.Template())
		t.Fail()
	}

	if snp.TabStopSize() != 4 {
		t.Fail()
	}

	ts := snp.TabStopAt(0)
	if ts.idx != 1 || ts.placeholder != "array" {
		t.Logf("wrong tabstop: %v", ts)
		t.Fail()
	}

	ts = snp.TabStopAt(1)
	if ts.idx != 2 || ts.placeholder != "element" {
		t.Logf("wrong tabstop: %v", ts)
		t.Fail()
	}

	ts = snp.TabStopAt(2)
	if ts.idx != 0 || ts.variable != "TM_CURRENT_LINE" || ts.variableDefault != "" {
		t.Logf("wrong tabstop: %v", ts)
		t.Fail()
	}

	ts = snp.TabStopAt(3)
	if !ts.IsFinal() {
		t.Logf("wrong tabstop: %v", ts)
		t.Fail()
	}

}
