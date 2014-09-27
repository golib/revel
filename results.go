package revel

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"code.google.com/p/go.net/websocket"
)

type Result interface {
	Apply(req *Request, resp *Response)
}

// Action methods return this result to request a template be rendered.
type RenderTemplateResult struct {
	Template     Template // this could be layout if exists
	TemplateName string   // of original template name, useful with error processing
	RenderArgs   map[string]interface{}
}

func (r *RenderTemplateResult) Apply(req *Request, resp *Response) {
	// Handle panics when rendering templates.
	defer func() {
		if err := recover(); err != nil {
			ERROR.Println(err)

			PlaintextErrorResult{fmt.Errorf("Template Execution Panic in %s:\n%s",
				r.Template.Name(), err)}.Apply(req, resp)
		}
	}()

	if captureErr := r.renderCapture(req, resp); captureErr != nil {
		ErrorResult{r.RenderArgs, captureErr}.Apply(req, resp)
		return
	}

	chunked := Config.BoolDefault("results.chunked", false)

	// If it's a HEAD request, throw away the bytes.
	out := io.Writer(resp.Out)
	if req.Method == "HEAD" {
		out = ioutil.Discard
	}

	// Render the template into a temporary buffer, to see if there was an error
	// rendering the template. If not, then copy it into the response buffer.
	// Otherwise, template render errors may result in unpredictable HTML (and
	// would carry a 200 status code)
	// (Always render to a temporary buffer first to avoid having error pages
	// distorted by HTML already written)
	var buf bytes.Buffer

	r.renderTemplate(req, resp, &buf)
	if buf.Len() == 0 {
		// avoid http connection hanging up!
		buf.WriteTo(out)
		return
	}

	resp.WriteHeader(http.StatusOK, "text/html; charset=utf-8")
	if !chunked {
		resp.Out.Header().Set("Content-Length", strconv.Itoa(buf.Len()))
	} else {
		// It isn't needed for progressive rendering.
		// However, it is needed when the total content length is unknown before the first bytes are sent.
		resp.Out.Header().Set("Transfer-Encoding", "chunked")
	}

	buf.WriteTo(out)
}

func (r *RenderTemplateResult) renderTemplate(req *Request, resp *Response, buf *bytes.Buffer) {
	err := r.Template.Render(buf, r.RenderArgs)
	if err == nil {
		return
	}

	// clean rendered result
	buf.Reset()

	var (
		content []string

		name, line, description = parseTemplateError(err)
	)
	if name == "" {
		name = r.Template.Name()
		content = r.Template.Content()
	} else if goTemplate, err := MainTemplateLoader.Template(name); err == nil {
		content = goTemplate.Content()
	}

	compileError := &Error{
		Title:       "Template Execution Error",
		Line:        line,
		Path:        name,
		Description: description,
		SourceLines: content,
	}

	ERROR.Printf("Template Execution Error (in %s): %s", name, description)

	// respond as internal server error
	resp.Status = 500

	ErrorResult{r.RenderArgs, compileError}.Apply(req, resp)
}

func (r *RenderTemplateResult) renderCapture(req *Request, resp *Response) error {
	yield2blocks, err := MainTemplateLoader.Yield2Blocks(r.TemplateName)
	if err != nil {
		return nil
	}

	// apply all yielded blocks to r.RenderArgs
	for yieldName, blockName := range yield2blocks {
		goTemplate, err := MainTemplateLoader.Template(blockName)
		if err != nil {
			ERROR.Println("Failed loading template block %s of %s : %s", blockName, r.TemplateName, err.Error())

			continue
		}

		captureResult := CaptureTemplateResult{
			Template:   goTemplate,
			RenderArgs: r.RenderArgs,
		}
		captureResult.Apply(req, resp)

		if blockErr := captureResult.Error(); blockErr != nil {
			if goTemplate, err := MainTemplateLoader.Template(r.TemplateName); err == nil {
				_, line, description := parseTemplateError(blockErr)

				return &Error{
					Title:       "Template Execution Error (Capture)",
					Line:        line,
					Path:        r.TemplateName,
					Description: description,
					SourceLines: goTemplate.Content(),
				}
			}

			return blockErr
		}

		r.RenderArgs[yieldName] = captureResult.HTML()
	}

	return nil
}

type CaptureTemplateResult struct {
	Template   Template
	RenderArgs map[string]interface{}

	renderedHtml  string
	renderedError error
}

func (r *CaptureTemplateResult) Apply(req *Request, resp *Response) {
	// Handle panics when rendering templates.
	defer func() {
		if panicErr := recover(); panicErr != nil {
			r.renderedError = errors.New(fmt.Sprintf("Template Execution Panic (Capture) : %s\n", panicErr))
			return
		}
	}()

	// Internal help template variable for some purposes?
	r.RenderArgs["RevelTemplateCaptureMode"] = true

	buf := bytes.NewBuffer([]byte{})

	if err := r.Template.Render(buf, r.RenderArgs); err != nil {
		r.renderedError = errors.New(fmt.Sprintf("Template Excution Error (Capture) : %s\n", err.Error()))
		return
	}

	r.renderedHtml = buf.String()
	return
}

func (r *CaptureTemplateResult) HTML() template.HTML {
	return template.HTML(r.renderedHtml)
}

func (r *CaptureTemplateResult) Error() error {
	return r.renderedError
}

type RenderTextResult struct {
	text string
}

func (r RenderTextResult) Apply(req *Request, resp *Response) {
	resp.WriteHeader(http.StatusOK, "text/plain; charset=utf-8")
	resp.Out.Write([]byte(r.text))
}

type RenderHtmlResult struct {
	html string
}

func (r RenderHtmlResult) Apply(req *Request, resp *Response) {
	resp.WriteHeader(http.StatusOK, "text/html; charset=utf-8")
	resp.Out.Write([]byte(r.html))
}

type RenderJsonResult struct {
	json     interface{}
	callback string
}

func (r RenderJsonResult) Apply(req *Request, resp *Response) {
	var (
		b   []byte
		err error
	)

	if Config.BoolDefault("results.pretty", false) {
		b, err = json.MarshalIndent(r.json, "", "  ")
	} else {
		b, err = json.Marshal(r.json)
	}

	if err != nil {
		ERROR.Println("json marshal error : ", err.Error())

		ErrorResult{Error: err}.Apply(req, resp)
		return
	}

	if r.callback == "" {
		resp.WriteHeader(http.StatusOK, "application/json; charset=utf-8")
		resp.Out.Write(b)
		return
	}

	resp.WriteHeader(http.StatusOK, "application/javascript; charset=utf-8")
	resp.Out.Write([]byte(r.callback + "("))
	resp.Out.Write(b)
	resp.Out.Write([]byte(");"))
}

type RenderXmlResult struct {
	xml interface{}
}

func (r RenderXmlResult) Apply(req *Request, resp *Response) {
	var (
		b   []byte
		err error
	)

	if Config.BoolDefault("results.pretty", false) {
		b, err = xml.MarshalIndent(r.xml, "", "  ")
	} else {
		b, err = xml.Marshal(r.xml)
	}

	if err != nil {
		ERROR.Println("xml marshal error : ", err.Error())

		ErrorResult{Error: err}.Apply(req, resp)
		return
	}

	resp.WriteHeader(http.StatusOK, "application/xml; charset=utf-8")
	resp.Out.Write(b)
}

// This result is used when the template loader or error template is not available.
type PlaintextErrorResult struct {
	Error error
}

func (r PlaintextErrorResult) Apply(req *Request, resp *Response) {
	resp.WriteHeader(http.StatusInternalServerError, "text/plain; charset=utf-8")
	resp.Out.Write([]byte(r.Error.Error()))
}

// This result handles all kinds of error codes (500, 404, ..).
// It renders the relevant error page (errors/CODE.format, e.g. errors/500.json).
// If RunMode is "dev", this results in a friendly error page.
type ErrorResult struct {
	RenderArgs map[string]interface{}
	Error      error
}

func (r ErrorResult) Apply(req *Request, resp *Response) {
	// This func shows a plaintext error message, in case the template rendering
	// doesn't work.
	showPlaintext := func(err error) {
		PlaintextErrorResult{fmt.Errorf("Server Error:\n%s\n\n"+
			"Additionally, an error occurred when rendering the error page:\n%s",
			r.Error, err)}.Apply(req, resp)
	}

	format := req.Format
	status := resp.Status
	if status == 0 {
		status = http.StatusInternalServerError
	}

	contentType := ContentTypeByFilename("revel." + format)
	if contentType == DefaultFileContentType {
		contentType = "text/plain"
	}

	// Get the error template.
	var err error
	templateName := fmt.Sprintf("errors/%d.%s", status, format)

	// first, search app/views/errors
	templateSet, err := MainTemplateLoader.Template(templateName)
	if err != nil || templateSet == nil {
		// second, fallback to revel templates/errors
		templateSet, err = RevelTemplateLoader.Template(templateName)
	}

	if templateSet == nil {
		if err == nil {
			err = fmt.Errorf("Couldn't find template %s", templateName)
		}

		showPlaintext(err)
		return
	}

	// If it's not a revel error, wrap it in one.
	var revelError *Error
	switch err := r.Error.(type) {
	case *Error:
		revelError = err
	case error:
		revelError = &Error{
			Title:       "Server Error",
			Description: err.Error(),
		}
	default:
		revelError = &Error{
			Title:       "Unknown Server Error",
			Description: "Unknown server error triggered",
		}
	}

	if r.RenderArgs == nil {
		r.RenderArgs = make(map[string]interface{})
	}
	r.RenderArgs["RunMode"] = RunMode
	r.RenderArgs["Error"] = revelError
	r.RenderArgs["Router"] = MainRouter

	// Render it.
	var buf bytes.Buffer
	err = templateSet.Render(&buf, r.RenderArgs)

	// If there was an error, print it in plain text.
	if err != nil {
		showPlaintext(err)
		return
	}

	// need to check if we are on a websocket here
	// net/http panics if we write to a hijacked connection
	if req.Method != "WS" {
		resp.WriteHeader(status, contentType)
		b.WriteTo(resp.Out)
	} else {
		websocket.Message.Send(req.Websocket, fmt.Sprint(revelError))
	}
}

type ContentDisposition string

var (
	Attachment ContentDisposition = "attachment"
	Inline     ContentDisposition = "inline"
)

type BinaryResult struct {
	Reader   io.Reader
	Name     string
	Length   int64
	Delivery ContentDisposition
	ModTime  time.Time
}

func (r *BinaryResult) Apply(req *Request, resp *Response) {
	disposition := string(r.Delivery)
	if r.Name != "" {
		disposition += fmt.Sprintf("; filename=%s", r.Name)
	}
	resp.Out.Header().Set("Content-Disposition", disposition)

	// If we have a ReadSeeker, delegate to http.ServeContent
	if rs, ok := r.Reader.(io.ReadSeeker); ok {
		// http.ServeContent doesn't know about response.ContentType, so we set the respective header.
		if resp.ContentType != "" {
			resp.Out.Header().Set("Content-Type", resp.ContentType)
		}
		http.ServeContent(resp.Out, req.Request, r.Name, r.ModTime, rs)
	} else {
		// Else, do a simple io.Copy.
		if r.Length != -1 {
			resp.Out.Header().Set("Content-Length", strconv.FormatInt(r.Length, 10))
		}
		resp.WriteHeader(http.StatusOK, ContentTypeByFilename(r.Name))
		io.Copy(resp.Out, r.Reader)
	}

	// Close the Reader if we can
	if v, ok := r.Reader.(io.Closer); ok {
		v.Close()
	}
}

type RedirectToUrlResult struct {
	url string
}

func (r *RedirectToUrlResult) Apply(req *Request, resp *Response) {
	resp.Out.Header().Set("Location", r.url)
	resp.WriteHeader(http.StatusFound, "")
}

type RedirectToActionResult struct {
	val  interface{}
	args map[string]string
}

func (r *RedirectToActionResult) Apply(req *Request, resp *Response) {
	url, err := FindResourceUrl(r.val, r.args)
	if err != nil {
		ERROR.Println("Couldn't resolve redirect:", err.Error())
		ErrorResult{Error: err}.Apply(req, resp)
		return
	}

	rurl := &RedirectToUrlResult{url: url}
	rurl.Apply(req, resp)
}
