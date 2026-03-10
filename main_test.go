package main

import "testing"

func TestBufColToVisualCol(t *testing.T) {
	tests := []struct {
		name    string
		line    []rune
		bufCol  int
		want    int
	}{
		{"no tabs", []rune("hello"), 3, 3},
		{"tab at start col0", []rune("\thello"), 0, 0},
		{"tab at start col1", []rune("\thello"), 1, 8},
		{"tab at start col2", []rune("\thello"), 2, 9},
		{"two tabs col2", []rune("\t\thello"), 2, 16},
		{"mid tab", []rune("ab\tcd"), 2, 2},
		{"after mid tab", []rune("ab\tcd"), 3, 8},
		{"after mid tab+1", []rune("ab\tcd"), 4, 9},
		{"empty line", []rune{}, 0, 0},
		{"col beyond line", []rune("ab"), 5, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := bufColToVisualCol(tt.line, tt.bufCol)
			if got != tt.want {
				t.Errorf("bufColToVisualCol(%q, %d) = %d, want %d", string(tt.line), tt.bufCol, got, tt.want)
			}
		})
	}
}
