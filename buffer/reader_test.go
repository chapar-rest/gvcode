package buffer

import (
	"testing"
	"unicode/utf8"
)

func TestRuneOffset(t *testing.T) {
	src := NewTextSource()
	src.Insert(0, "hello,world.")
	if src.RuneOffset(12) != 12 {
		t.Fail()
	}

	src.Insert(12, "你好，世界")

	_, size := utf8.DecodeRuneInString("你好，世界")
	t.Log(size)
	if src.RuneOffset(13) != 15 {
		t.Fail()
	}
}

func TestReadAt(t *testing.T) {
	src := NewTextSource()
	src.Insert(0, "hello,world.")

	if src.Len() != 12 {
		t.Fail()
	}

	buf := make([]byte, 5)
	n, err := src.ReadAt(buf, 0)
	if err != nil {
		t.Fail()
	}

	if n != 5 || string(buf) != "hello" {
		t.Fail()
	}

	content := src.Text(buf)
	if string(content) != "hello,world." {
		t.Fail()
	}
}

func TestReadRuneAt(t *testing.T) {
	src := NewTextSource()
	src.SetText([]byte("hello,world.你好，世界"))

	r, err := src.ReadRuneAt(6)
	if err != nil {
		t.Logf("ReadRuneAt error: %v", err)
		t.Fail()
	}
	if r != 'w' {
		t.Fail()
	}

	r, err = src.ReadRuneAt(12)
	if err != nil {
		t.Logf("ReadRuneAt error: %v", err)
		t.Fail()
	}
	if r != '你' {
		t.Fail()
	}

}
