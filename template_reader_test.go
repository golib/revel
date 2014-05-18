package revel

import (
	"path"
	"strings"
	"testing"
)

var (
	layout  = path.Join(RevelPath, "testdata", "views", "layout.html")
	content = path.Join(RevelPath, "testdata", "views", "content.html")
)

func TestTemplateReader(t *testing.T) {
	tr, _ := NewTemplateReader(layout)
	tr.Parse()

	layout_byname := tr.byname([]string{""})
	if !strings.Contains(tr.Template, layout_byname) {
		t.Errorf("Expected block contains `%s`, but got %s", layout_byname, tr.Template)
	}

	tr, _ = NewTemplateReader(content)
	tr.Parse()

	content_byname := tr.byname([]string{""})
	if strings.Contains(tr.Template, content_byname) {
		t.Errorf("Expected block doesn't contain `%s`, but got %s", content_byname, tr.Template)
	}
}

func TestTemplateReaderWithYields(t *testing.T) {
	tr, _ := NewTemplateReader(layout)
	tr.Parse()

	layout_byname := tr.byname([]string{""})

	if len(tr.Yields) != 2 {
		t.Errorf("Expected invoking yield 2 times, but got %d", len(tr.Yields))
	}

	if _, ok := tr.Yields[layout_byname]; !ok {
		t.Errorf("Expected contains yield `%s`, but got %#v", layout_byname, tr.Yields)
	}

	if len(tr.Blocks) != 1 {
		t.Errorf("Expected 1 block, but got %d", len(tr.Blocks))
	}

	if _, ok := tr.Blocks[layout_byname]; !ok {
		t.Errorf("Expected contains block `%s`, but got %#v", layout_byname, tr.Blocks)
	}

	layout_byname_var := "{{." + layout_byname + "}}"
	if html, _ := tr.Blocks[layout_byname]; !strings.Contains(html, layout_byname_var) {
		t.Errorf("Expected block contains `%s`, but got %s", layout_byname_var, html)
	}
}

func TestTemplateReaderWithBlocks(t *testing.T) {
	tr, _ := NewTemplateReader(content)
	tr.Parse()

	content_byname := tr.byname([]string{""})

	if len(tr.Yields) != 3 {
		t.Errorf("Expected invoking yield 3 times, but got %d", len(tr.Yields))
	}

	if _, ok := tr.Yields[content_byname]; !ok {
		t.Errorf("Expected contains yield `%s`, but got %#v", content_byname, tr.Yields)
	}

	if len(tr.Blocks) != 3 {
		t.Errorf("Expected 3 blocks, but got %d", len(tr.Blocks))
	}

	if _, ok := tr.Blocks[content_byname]; !ok {
		t.Errorf("Expected contains block `%s`, but got %#v", content_byname, tr.Blocks)
	}

	if html, _ := tr.Blocks[content_byname]; !strings.Contains(html, "This is normal template content") {
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
