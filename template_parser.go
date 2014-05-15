package revel

import (
	"bufio"
	"crypto/sha1"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

var (
	ryield  = regexp.MustCompile(`\A\s*=\s*yield\s*?\z|\A\s*=\s*yield\s+"([a-zA-Z]+)"\s*?\z|#{\s*yield\s+"([a-zA-Z]+)"\s*?}|\A\s*=\s*yield\s*\(\s*"([a-zA-Z]+)"\s*\)\s*?\z|#{\s*yield\s*\(\s*"([a-zA-Z]+)"\s*\)\s*?}`)
	rblock  = regexp.MustCompile(`\A-\s+block\s+"([a-zA-Z]+)"\s*?\z|\A-\s+block\s*\(\s*"([a-zA-Z]+)"\s*\)\s*?\z`)
	rindent = regexp.MustCompile(`\A[ \t]+?`)
)

type TemplateReader struct {
	Yields map[string]string
	Blocks map[string]string

	fp     *os.File
	reader *bufio.Reader
	line   string
	file   string
}

func NewTemplateReader(file string) *TemplateReader {
	fp, err := os.Open(file)
	if err != nil {
		return nil
	}

	return &TemplateReader{
		Yields: map[string]string{},
		Blocks: map[string]string{},

		fp:     fp,
		reader: bufio.NewReader(fp),
		line:   "",
		file:   file,
	}
}

func (tr *TemplateReader) Parse() {
	defer tr.Close()

	lines := []string{}
	for tr.readline() {
		switch {
		case tr.line[0] == '-': // block define
			line := tr.consumeline()

			matches := rblock.FindStringSubmatch(line)
			if len(matches) != 3 {
				ERROR.Panicf("Unexpected block syntax: %s", line)
			}

			// generate unique block name
			byName := tr.byname(matches[1:])

			// ensure next line is indented at least on white space
			tr.readline()
			if !rindent.MatchString(tr.line) {
				ERROR.Panicf("Unexpected terminate of block")
			}

			blockLines := []string{tr.consumeline()}
			for tr.readline() {
				if !rindent.MatchString(tr.line) {
					break
				}

				blockLines = append(blockLines, tr.consumeline())
			}

			tr.Blocks[byName] = strings.Join(blockLines, "\n")
		default:
			line := tr.consumeline()

			matches := ryield.FindStringSubmatch(line)
			if len(matches) == 5 {
				// generate unique block name
				byName := tr.byname(matches[1:])

				tr.Yields[byName] = fmt.Sprintf("{{.%s}}", byName)

				line = ryield.ReplaceAllString(line, tr.Yields[byName])
			}

			lines = append(lines, line)
		}
	}

	tr.Blocks[tr.byname([]string{""})] = strings.Join(lines, "\n")
	return
}

func (tr *TemplateReader) Close() {
	tr.fp.Close()
	return
}

func (tr *TemplateReader) readline() bool {
	// un-consumed line
	if tr.line != "" {
		return true
	}

	line, err := tr.reader.ReadString('\n')
	if err != nil {
		if err != io.EOF {
			ERROR.Fatalln("Failed template readline : ", err.Error())
			return false
		}

		// end of file
		tr.line = ""
		return false
	}

	tr.line = line
	return true
}

func (tr *TemplateReader) consumeline() string {
	line := tr.line
	if line == "" {
		ERROR.Fatal("Unexpected template consumeline")
	}

	tr.line = ""

	return line
}

// block / yield name generator
func (tr *TemplateReader) byname(names []string) string {
	var (
		s    = "__revel_default_yield_or_block__"
		name = strings.Join(names, "")
	)

	if name != "" {
		s = fmt.Sprintf("%s#%s", tr.file, name)
		s = strings.ToLower(s)
		s = strings.Replace(s, `/`, `_`, -1)
		s = strings.Replace(s, `\`, `_`, -1)
	}

	sha := sha1.New()
	io.WriteString(sha, s)

	return fmt.Sprintf("revel_%x", sha.Sum([]byte("revel")))
}
