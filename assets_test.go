package main

import (
	"bytes"
	"os"
	"reflect"
	"testing"
)

// TestLoadFrames verifies LoadFrames returns exactly four non-empty frames
// whose content is byte-identical to the on-disk ASCII art files, in load
// order: idle pose first, then tolk0, tolk1, tolk2.
func TestLoadFrames(t *testing.T) {
	frames, err := LoadFrames()
	if err != nil {
		t.Fatalf("LoadFrames() returned error: %v", err)
	}
	if len(frames) != 4 {
		t.Fatalf("LoadFrames() returned %d frames, want 4", len(frames))
	}

	expected := []struct {
		name string
		file string
	}{
		{"idle", "ascii_art/idle.txt"},
		{"tolk0", "ascii_art/tolk0.txt"},
		{"tolk1", "ascii_art/tolk1.txt"},
		{"tolk2", "ascii_art/tolk2.txt"},
	}

	for i, exp := range expected {
		t.Run(exp.name, func(t *testing.T) {
			if len(frames[i]) == 0 {
				t.Fatalf("frame[%d] (%s) is empty", i, exp.file)
			}

			disk, err := os.ReadFile(exp.file)
			if err != nil {
				t.Fatalf("os.ReadFile(%s) error: %v", exp.file, err)
			}

			embedded, err := framesFS.ReadFile(exp.file)
			if err != nil {
				t.Fatalf("framesFS.ReadFile(%s) error: %v", exp.file, err)
			}

			if !bytes.Equal(embedded, disk) {
				t.Errorf("embedded %s is not byte-identical to on-disk file", exp.file)
			}

			want := splitLines(string(disk))
			if !reflect.DeepEqual(frames[i], want) {
				t.Errorf("frame[%d] (%s) does not match file content line-for-line", i, exp.file)
			}
		})
	}
}

// TestSplitLines covers splitLines edge cases: CRLF endings stripped, a single
// trailing newline removed, and empty content returning an empty slice.
func TestSplitLines(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"CRLF endings stripped", "a\r\nb\r\n", []string{"a", "b"}},
		{"CRLF without trailing newline", "a\r\nb", []string{"a", "b"}},
		{"single trailing newline removed", "a\nb\n", []string{"a", "b"}},
		{"empty content returns empty slice", "", []string{}},
		{"no trailing newline", "a\nb", []string{"a", "b"}},
		{"single line without newline", "hello", []string{"hello"}},
		{"single line with trailing newline", "hello\n", []string{"hello"}},
		{"single line CRLF", "hello\r\n", []string{"hello"}},
		{"multiple trailing newlines keep one empty line", "a\n\n", []string{"a", ""}},
		{"only newline", "\n", []string{""}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitLines(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("splitLines(%q) = %#v, want %#v", tt.input, got, tt.want)
			}
		})
	}
}