package main

import (
	"testing"
)

func TestParseGrepLine(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    GrepResult
		wantOK  bool
	}{
		{
			name:   "standard line",
			input:  "main.go:10:func main() {",
			want:   GrepResult{File: "main.go", Line: 10, Text: "func main() {"},
			wantOK: true,
		},
		{
			name:   "colons in text",
			input:  "main.go:10:foo := bar:baz",
			want:   GrepResult{File: "main.go", Line: 10, Text: "foo := bar:baz"},
			wantOK: true,
		},
		{
			name:   "empty text portion",
			input:  "file.go:1:",
			want:   GrepResult{File: "file.go", Line: 1, Text: ""},
			wantOK: true,
		},
		{
			name:   "path with directory",
			input:  "src/pkg/file.go:42:some text",
			want:   GrepResult{File: "src/pkg/file.go", Line: 42, Text: "some text"},
			wantOK: true,
		},
		{
			name:   "spaces in file path",
			input:  "my file.go:5:hello world",
			want:   GrepResult{File: "my file.go", Line: 5, Text: "hello world"},
			wantOK: true,
		},
		{
			name:   "empty line",
			input:  "",
			wantOK: false,
		},
		{
			name:   "no colons",
			input:  "just some text",
			wantOK: false,
		},
		{
			name:   "one colon only",
			input:  "file.go:notanumber",
			wantOK: false,
		},
		{
			name:   "non-numeric line number",
			input:  "file.go:abc:text",
			wantOK: false,
		},
		{
			name:   "binary file notice",
			input:  "Binary file matches",
			wantOK: false,
		},
		{
			name:   "header line with dashes",
			input:  "--",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ParseGrepLine(tt.input)
			if ok != tt.wantOK {
				t.Errorf("ParseGrepLine(%q) ok = %v, want %v", tt.input, ok, tt.wantOK)
				return
			}
			if ok && got != tt.want {
				t.Errorf("ParseGrepLine(%q) = %+v, want %+v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseGrepOutput(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int // expected number of results
	}{
		{
			name:  "multiple valid lines",
			input: "a.go:1:hello\nb.go:2:world\nc.go:3:foo",
			want:  3,
		},
		{
			name:  "mixed valid and invalid",
			input: "a.go:1:hello\n\nBinary file matches\nb.go:2:world",
			want:  2,
		},
		{
			name:  "all invalid",
			input: "no match\n\n--",
			want:  0,
		},
		{
			name:  "empty input",
			input: "",
			want:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := ParseGrepOutput(tt.input)
			if len(results) != tt.want {
				t.Errorf("ParseGrepOutput() returned %d results, want %d", len(results), tt.want)
			}
		})
	}

	// Verify specific parsing of multi-line output
	t.Run("correct parsing of fields", func(t *testing.T) {
		output := "main.go:10:foo := bar:baz\nsrc/util.go:20:func helper() {"
		results := ParseGrepOutput(output)
		if len(results) != 2 {
			t.Fatalf("expected 2 results, got %d", len(results))
		}
		if results[0].File != "main.go" || results[0].Line != 10 || results[0].Text != "foo := bar:baz" {
			t.Errorf("result[0] = %+v, unexpected", results[0])
		}
		if results[1].File != "src/util.go" || results[1].Line != 20 || results[1].Text != "func helper() {" {
			t.Errorf("result[1] = %+v, unexpected", results[1])
		}
	})
}
