package output

import (
	"testing"
)

func TestIsBinary_NullByte(t *testing.T) {
	data := []byte("hello\x00world")
	if !IsBinary(data) {
		t.Error("expected true for data with null byte")
	}
}

func TestIsBinary_PlainText(t *testing.T) {
	data := []byte("hello world\nfoo bar\n")
	if IsBinary(data) {
		t.Error("expected false for plain text")
	}
}

func TestIsBinary_ValidUTF8(t *testing.T) {
	data := []byte("café résumé naïve 日本語")
	if IsBinary(data) {
		t.Error("expected false for valid UTF-8")
	}
}

func TestIsBinary_InvalidUTF8(t *testing.T) {
	// 0xff 0xfe is not valid UTF-8
	data := []byte{0x68, 0x65, 0xff, 0xfe, 0x6c, 0x6f}
	if !IsBinary(data) {
		t.Error("expected true for invalid UTF-8 bytes")
	}
}

func TestIsBinary_Empty(t *testing.T) {
	if IsBinary([]byte{}) {
		t.Error("expected false for empty data")
	}
}

func TestIsBinary_JSONContent(t *testing.T) {
	data := []byte(`{"key":"value","num":42}`)
	if IsBinary(data) {
		t.Error("expected false for JSON content")
	}
}

func TestIsBinary_HighNonPrintableRatio(t *testing.T) {
	// Simulate PNG-like header with many non-printable control bytes
	data := make([]byte, 100)
	for i := range data {
		data[i] = byte(i % 8) // lots of control chars 0x00-0x07
	}
	if !IsBinary(data) {
		t.Error("expected true for data with high non-printable ratio")
	}
}

func TestIsBinary_NewlinesAndTabs(t *testing.T) {
	data := []byte("line1\nline2\tcolumn\r\nline3\n")
	if IsBinary(data) {
		t.Error("expected false for text with newlines and tabs")
	}
}
