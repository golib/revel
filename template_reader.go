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
	Template     string            // of default block
	Blocks       map[string]string // of all blocks key => template
	Yield2Blocks map[string]string // of all keys yield => block

	fp     *os.File
	reader *bufio.Reader
	line   string
	file   string
}

func NewTemplateReader(file string) (*TemplateReader, error) {
	fp, err := os.Open(file)
	if err != nil {
		return nil, err
	}

	tr := &TemplateReader{
		Template:     "",
		Blocks:       map[string]string{},
		Yield2Blocks: map[string]string{},

		fp:     fp,
		reader: bufio.NewReader(fp),
		line:   "",
		file:   file,
	}

	return tr, nil
}

func (tr *TemplateReader) Parse() {
	defer tr.Close()

	lines := []string{}
	for tr.readline() {
		switch {
		case tr.line[0] == '-' && rblock.MatchString(tr.line): // block define
			line := tr.consumeline()

			matches := rblock.FindStringSubmatch(line)
			if len(matches) != 3 {
				ERROR.Panicf("Unexpected block syntax: %s", line)
			}

			// ensure next line is indented at least one white space
			tr.readline()
			if !rindent.MatchString(tr.line) {
				ERROR.Panicf("Unexpected terminate of block")
			}

			// generate unique block name
			name := tr.name(matches[1:])
			yieldName := tr.yieldName(name)
			blockName := tr.blockName(name)

			blockLines := []string{tr.consumeline()}
			for tr.readline() {
				if strings.TrimSpace(tr.line) != "" && !rindent.MatchString(tr.line) {
					break
				}

				blockLines = append(blockLines, tr.consumeline())
			}

			tr.Blocks[blockName] = strings.Join(blockLines, "\n")
			tr.Yield2Blocks[yieldName] = blockName
		default:
			line := tr.consumeline()

			matches := ryield.FindStringSubmatch(line)
			if len(matches) == 5 {
				// generate unique block name
				name := tr.name(matches[1:])
				yieldName := tr.yieldName(name)

				line = ryield.ReplaceAllString(line, fmt.Sprintf("{{.%s}}", yieldName))
			}

			lines = append(lines, line)
		}
	}

	// default block
	tr.Template = strings.Join(lines, "\n")

	// trick of default block
	yieldName := tr.yieldName("")
	blockName := tr.blockName("")
	tr.Blocks[blockName] = tr.Template
	tr.Yield2Blocks[yieldName] = blockName
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

func (tr *TemplateReader) name(names []string) string {
	name := strings.Join(names, "")
	name = strings.ToLower(name)
	return strings.TrimSpace(name)
}

func (tr *TemplateReader) yieldName(name string) string {
	if name == "" {
		return "revel_yield_default_content"
	}

	name = strings.Replace(name, `/`, `_`, -1)
	name = strings.Replace(name, `\`, `_`, -1)

	return fmt.Sprintf("revel_yield_%s", name)
}

func (tr *TemplateReader) blockName(name string) string {
	name = fmt.Sprintf("%s#%s", tr.file, name)
	name = strings.Replace(name, `/`, `_`, -1)
	name = strings.Replace(name, `\`, `_`, -1)

	sha := sha1.New()
	io.WriteString(sha, name)

	return fmt.Sprintf("revel_block_%x", sha.Sum([]byte("revel")))
}
