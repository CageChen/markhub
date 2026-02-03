package markdown

import (
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	p := NewParser()
	source := []byte("# Hello World\n\nThis is a *test*.")

	result, err := p.Parse(source)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if !strings.Contains(result.HTML, "<h1") || !strings.Contains(result.HTML, "Hello World</h1>") {
		t.Error("expected H1 tag containing 'Hello World' in HTML")
	}
	if !strings.Contains(result.HTML, "<em>test</em>") {
		t.Error("expected italicized test in HTML")
	}
	if result.Title != "Hello World" {
		t.Errorf("expected title Hello World, got %s", result.Title)
	}
}

func TestExtractTOC(t *testing.T) {
	p := NewParser()
	source := []byte("# Head 1\n## Head 2\n### Head 3")

	toc := p.extractTOC(source)
	if len(toc) != 3 {
		t.Fatalf("expected 3 TOC items, got %d", len(toc))
	}

	if toc[0].Level != 1 || toc[0].Title != "Head 1" {
		t.Errorf("TOC item 0 mismatch: %+v", toc[0])
	}
	if toc[1].Level != 2 || toc[1].Title != "Head 2" {
		t.Errorf("TOC item 1 mismatch: %+v", toc[1])
	}
	if toc[2].Level != 3 || toc[2].Title != "Head 3" {
		t.Errorf("TOC item 2 mismatch: %+v", toc[2])
	}
}

func TestGenerateAnchor(t *testing.T) {
	tests := []struct {
		input  string
		output string
	}{
		{"Hello World", "hello-world"},
		{"Test! @# Content", "test-content"},
		{"Multiple   Spaces", "multiple-spaces"},
		{"-Start-and-End-", "start-and-end"},
		{"中文标题", "中文标题"},
	}

	for _, tt := range tests {
		got := generateAnchor(tt.input)
		if got != tt.output {
			t.Errorf("generateAnchor(%q) = %q, want %q", tt.input, got, tt.output)
		}
	}
}
