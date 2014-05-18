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

	titleYieldName := tr.yieldName("title")
	defaultYieldName := tr.yieldName("")

	if !strings.Contains(tr.Template, titleYieldName) {
		t.Errorf("Expected layout contains `%s`, but got %s", titleYieldName, tr.Template)
	}

	if !strings.Contains(tr.Template, defaultYieldName) {
		t.Errorf("Expected layout contains `%s`, but got %s", defaultYieldName, tr.Template)
	}

	tr, _ = NewTemplateReader(content)
	tr.Parse()

	titleBlockName := tr.Yield2Blocks[titleYieldName]
	defaultBlockName := tr.Yield2Blocks[defaultYieldName]

	if _, ok := tr.Blocks[titleBlockName]; !ok {
		t.Errorf("Expected block contains `%s`, but got %#v", titleBlockName, tr.Blocks)
	}

	if _, ok := tr.Blocks[defaultBlockName]; !ok {
		t.Errorf("Expected block contains `%s`, but got %#v", defaultBlockName, tr.Blocks)
	}
}

func TestTemplateReaderWithYields(t *testing.T) {
	tr, _ := NewTemplateReader(layout)
	tr.Parse()

	if len(tr.Yield2Blocks) != 1 {
		t.Errorf("Expected invoking yield %d times, but got %d", 1, len(tr.Yield2Blocks))
	}

	if len(tr.Blocks) != 1 {
		t.Errorf("Expected 1 block, but got %d", len(tr.Blocks))
	}

	if !strings.Contains(tr.Template, tr.Yield2Blocks[tr.yieldName("title")]) {
		t.Errorf("Expected yield `%s`, but got %s", tr.yieldName("title"), tr.Template)
	}

	if !strings.Contains(tr.Template, tr.yieldName("")) {
		t.Errorf("Expected yield `%s`, but got %s", tr.yieldName(""), tr.Template)
	}
}

func TestTemplateReaderWithBlocks(t *testing.T) {
	tr, _ := NewTemplateReader(content)
	tr.Parse()

	if len(tr.Yield2Blocks) != 3 {
		t.Errorf("Expected invoking yield 3 times, but got %d", len(tr.Yield2Blocks))
	}

	if len(tr.Blocks) != 3 {
		t.Errorf("Expected 3 blocks, but got %d", len(tr.Blocks))
	}

	if !strings.Contains(tr.Blocks[tr.blockName("title")], "Layout template title") {
		t.Errorf("Expected title block contains `%s`, but got %s",
			"Layout template title",
			tr.Blocks[tr.blockName("title")])
	}

	if !strings.Contains(tr.Blocks[tr.blockName("content")], "This is a layout template content") {
		t.Errorf("Expected content block contains `%s`, but got %s",
			"This is a layout template content",
			tr.Blocks[tr.blockName("content")])
	}

	if !strings.Contains(tr.Blocks[tr.blockName("")], "This is normal template content") {
		t.Errorf("Expected default block contains `%s`, but got %s",
			"This is normal template content",
			tr.Blocks[tr.blockName("")])
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
