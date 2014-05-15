package revel

import (
	"path"
	"strings"
	"testing"
)

var (
	layout  = path.Join(RevelPath, "testdata", "views", "layout.html")
	content = path.Join(RevelPath, "testdata", "views", "content.html")

	yield_or_block     = "revel_726576656c2fa43a078b3629f7fd390756c3edf515b0ae9b9b"
	yield_or_block_var = "{{.revel_726576656c2fa43a078b3629f7fd390756c3edf515b0ae9b9b}}"
)

func TestTemplateReader(t *testing.T) {
	tr, _ := NewTemplateReader(layout)
	tr.Parse()

	if !strings.Contains(tr.Template, yield_or_block_var) {
		t.Errorf("Expected block contains `%s`, but got %s", yield_or_block_var, tr.Template)
	}

	tr, _ = NewTemplateReader(content)
	tr.Parse()

	if strings.Contains(tr.Template, yield_or_block_var) {
		t.Errorf("Expected block doesn't contain `%s`, but got %s", yield_or_block_var, tr.Template)
	}
}

func TestTemplateReaderWithYields(t *testing.T) {
	tr, _ := NewTemplateReader(layout)
	tr.Parse()

	if len(tr.Yields) != 2 {
		t.Errorf("Expected invoking yield 2 times, but got %d", len(tr.Yields))
	}

	if _, ok := tr.Yields[yield_or_block]; !ok {
		t.Errorf("Expected contains yield `%s`, but nothing found", yield_or_block)
	}

	if len(tr.Blocks) != 1 {
		t.Errorf("Expected 1 block, but got %d", len(tr.Blocks))
	}

	if _, ok := tr.Blocks[yield_or_block]; !ok {
		t.Errorf("Expected contains block `%s`, but nothing found", yield_or_block)
	}

	if html, _ := tr.Blocks[yield_or_block]; !strings.Contains(html, yield_or_block_var) {
		t.Errorf("Expected block contains `%s`, but got %s", yield_or_block_var, html)
	}
}

func TestTemplateReaderWithBlocks(t *testing.T) {
	tr, _ := NewTemplateReader(content)
	tr.Parse()

	if len(tr.Yields) != 3 {
		t.Errorf("Expected invoking yield 3 times, but got %d", len(tr.Yields))
	}

	if _, ok := tr.Yields[yield_or_block]; !ok {
		t.Errorf("Expected contains yield `%s`, but nothing found", yield_or_block)
	}

	if len(tr.Blocks) != 3 {
		t.Errorf("Expected 3 blocks, but got %d", len(tr.Blocks))
	}

	if _, ok := tr.Blocks[yield_or_block]; !ok {
		t.Errorf("Expected contains block `%s`, but nothing found", yield_or_block)
	}

	if html, _ := tr.Blocks[yield_or_block]; !strings.Contains(html, "This is normal template content") {
		t.Errorf("Expected block contains `%s`, but got %s", "This is normal template content", html)
	}
}

func BenchmarkTemplateReaderWithYields(b *testing.B) {
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		tr, _ := NewTemplateReader(layout)
		tr.Parse()
	}
}

func BenchmarkTemplateReaderWithBlocks(b *testing.B) {
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		tr, _ := NewTemplateReader(content)
		tr.Parse()
	}
}
