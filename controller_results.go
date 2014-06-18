package revel

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func (c *Controller) LayoutName() string {
	layoutName := ""

	layouter, ok := c.AppController.(ControllerLayouter)
	if !ok {
		return layoutName
	}

	layouts := layouter.Layout()

	// is there an action layout defined?
	if layout, ok := layouts[c.MethodName]; ok {
		layoutName = layout

		// is there an method:action layout defined?
	} else if layout, ok := layouts[c.Request.Method+":"+c.MethodName]; ok {
		layoutName = layout

		// is there an wildcard layout defined?
	} else if layout, ok := layouts["*"]; ok {
		layoutName = layout
	}

	return strings.TrimSpace(layoutName)
}

// Perform a message lookup for the given message name using the given arguments
// using the current language defined for this controller.
//
// The current language is set by the i18n plugin.
func (c *Controller) Message(message string, args ...interface{}) (value string) {
	return Message(c.Request.Locale, message, args...)
}

func (c *Controller) SetCookie(cookie *http.Cookie) {
	http.SetCookie(c.Response.Out, cookie)
}

// FlashParams serializes the contents of Controller.Params to the Flash
// cookie.
func (c *Controller) FlashParams() {
	for key, vals := range c.Params.Values {
		c.Flash.Out[key] = strings.Join(vals, ",")
	}
}

// Render a template corresponding to the calling Controller method.
// Arguments will be added to c.RenderArgs prior to rendering the template.
// They are keyed on their local identifier.
//
// For example:
//
//     func (c Users) ShowUser(id int) revel.Result {
//       user := loadUser(id)
//       return c.Render(user)
//     }
//
// This action will render views/Users/ShowUser.html, passing in an extra
// key-value "user": (User).
func (c *Controller) Render(extraRenderArgs ...interface{}) Result {
	// Get the calling function name.
	_, _, line, ok := runtime.Caller(1)
	if !ok {
		ERROR.Println("Failed to get Caller information")
	}

	// Get the extra RenderArgs passed in.
	if renderArgNames, ok := c.MethodType.RenderArgNames[line]; ok {
		if len(renderArgNames) == len(extraRenderArgs) {
			for i, extraRenderArg := range extraRenderArgs {
				c.RenderArgs[renderArgNames[i]] = extraRenderArg
			}
		} else {
			ERROR.Println(len(renderArgNames), "RenderArg names found for",
				len(extraRenderArgs), "extra RenderArgs")
		}
	} else {
		ERROR.Println("No RenderArg names found for Render call on line", line,
			"(Action", c.Action, ")")
	}

	return c.RenderTemplate(c.Name + "/" + c.MethodType.Name + "." + c.Request.Format)
}

// A less magical way to render a template.
// Renders the given template, using the current RenderArgs.
func (c *Controller) RenderTemplate(name string) Result {
	// always using lowercase
	name = strings.ToLower(name)

	// layout?
	templateName := name
	if layoutName := c.LayoutName(); layoutName != "" {
		templateName = layoutName
	}

	// Get the Template.
	goTemplate, err := MainTemplateLoader.Template(templateName)
	if err != nil {
		return c.RenderError(err)
	}

	return &RenderTemplateResult{
		Template:     goTemplate,
		TemplateName: name,
		RenderArgs:   c.RenderArgs,
	}
}

// Capture a template corresponding to the calling Controller method.
//
// For example:
//
//     func (c Users) ShowUser(id int) revel.Result {
//       user := loadUser(id)
//       return c.Capture()
//     }
//
// This action will render views/Users/ShowUser.html,
// capture the result and return the result as template.HTML.
func (c *Controller) Capture(name string) template.HTML {
	return c.CaptureTemplate(c.Name + "/" + c.MethodType.Name + "." + c.Request.Format)
}

// A less magical way to capture a template.
// Renders the given template, using the current RenderArgs.
// TODO: refactor this with RenderTemplateResult.renderCapture!
func (c *Controller) CaptureTemplate(name string) template.HTML {
	// always using lowercase
	name = strings.ToLower(name)

	// layout?
	templateName := name
	if layoutName := c.LayoutName(); layoutName != "" {
		templateName = layoutName
	}

	// Get the Template.
	goTemplate, err := MainTemplateLoader.Template(templateName)
	if err != nil {
		ERROR.Printf("Load capture template %s with error : %s\n", templateName, err)

		return ""
	}

	// apply blocks
	yield2blocks, err := MainTemplateLoader.Yield2Blocks(templateName)
	if err != nil {
		ERROR.Printf("Load capture template blocks %s with error : %s\n", templateName, err)

		return ""
	}

	// apply all yielded blocks to r.RenderArgs
	for yieldName, blockName := range yield2blocks {
		blockTemplate, err := MainTemplateLoader.Template(blockName)
		if err != nil {
			ERROR.Println("Failed loading template block %s of %s : %s", blockName, templateName, err.Error())

			continue
		}

		captureResult := CaptureTemplateResult{
			Template:   blockTemplate,
			RenderArgs: c.RenderArgs,
		}
		captureResult.Apply(c.Request, c.Response)

		if blockErr := captureResult.Error(); blockErr != nil {
			ERROR.Printf("Render capture block template %s with error : %s\n", blockName, err)

			return ""
		}

		c.RenderArgs[yieldName] = captureResult.HTML()
	}

	captureResult := &CaptureTemplateResult{
		Template:   goTemplate,
		RenderArgs: c.RenderArgs,
	}
	captureResult.Apply(c.Request, c.Response)

	if captureErr := captureResult.Error(); captureErr != nil {
		ERROR.Printf("Capture template %s with error : %s\n", templateName, captureErr)

		return ""
	}

	return captureResult.HTML()
}

// Render an error in request.Format
func (c *Controller) RenderError(err error) Result {
	return ErrorResult{c.RenderArgs, err}
}

// Render plaintext in response, printf style.
func (c *Controller) RenderText(text string, args ...interface{}) Result {
	s := text
	if len(args) > 0 {
		s = fmt.Sprintf(text, args...)
	}

	return &RenderTextResult{s}
}

// Uses encoding/json.Marshal to return JSON to the client.
func (c *Controller) RenderJson(json interface{}) Result {
	return RenderJsonResult{json, ""}
}

// Renders a JSONP result using encoding/json.Marshal
func (c *Controller) RenderJsonP(callback string, json interface{}) Result {
	return RenderJsonResult{json, callback}
}

// Uses encoding/xml.Marshal to return XML to the client.
func (c *Controller) RenderXml(xml interface{}) Result {
	return RenderXmlResult{xml}
}

// Render html in response
func (c *Controller) RenderHtml(html string) Result {
	return &RenderHtmlResult{html}
}

// RenderFile returns a file, either displayed inline or downloaded
// as an attachment. The name and size are taken from the file info.
func (c *Controller) RenderFile(file *os.File, delivery ContentDisposition) Result {
	var (
		modtime       = time.Now()
		fileInfo, err = file.Stat()
	)

	if err != nil {
		WARN.Println("RenderFile error:", err)
	}

	if fileInfo != nil {
		modtime = fileInfo.ModTime()
	}

	return c.RenderBinary(file, filepath.Base(file.Name()), delivery, modtime)
}

// RenderBinary is like RenderFile() except that it instead of a file on disk,
// it renders data from memory (which could be a file that has not been written,
// the output from some function, or bytes streamed from somewhere else, as long
// it implements io.Reader).  When called directly on something generated or
// streamed, modtime should mostly likely be time.Now().
func (c *Controller) RenderBinary(memfile io.Reader, filename string, delivery ContentDisposition, modtime time.Time) Result {
	return &BinaryResult{
		Reader:   memfile,
		Name:     filename,
		Delivery: delivery,
		Length:   -1, // http.ServeContent gets the length itself unless memfile is a stream.
		ModTime:  modtime,
	}
}

// Redirect to an action or to a URL.
//   c.Redirect(Controller.Action)
//   c.Redirect("/controller/action")
//   c.Redirect("/controller/%d/action", id)
func (c *Controller) Redirect(val interface{}, args ...interface{}) Result {
	if url, ok := val.(string); ok {
		if len(args) == 0 {
			return &RedirectToUrlResult{url}
		}

		return &RedirectToUrlResult{fmt.Sprintf(url, args...)}
	}

	actionArgs := map[string]string{}
	if len(args) > 0 {
		if arg, ok := args[0].(map[string]string); ok {
			actionArgs = arg
		}
	}

	return &RedirectToActionResult{val, actionArgs}
}

// Forbidden returns an HTTP 403 Forbidden response whose body is the
// formatted string of msg and args.
func (c *Controller) Forbidden(msg string, args ...interface{}) Result {
	s := msg
	if len(args) > 0 {
		s = fmt.Sprintf(msg, args...)
	}

	c.Response.Status = http.StatusForbidden

	return c.RenderError(&Error{
		Title:       "Forbidden",
		Description: s,
	})
}

// NotFound returns an HTTP 404 Not Found response whose body is the
// formatted string of msg and args.
func (c *Controller) NotFound(msg string, args ...interface{}) Result {
	s := msg
	if len(args) > 0 {
		s = fmt.Sprintf(msg, args...)
	}

	c.Response.Status = http.StatusNotFound

	return c.RenderError(&Error{
		Title:       "Not Found",
		Description: s,
	})
}

// NotImplemented returns an HTTP 501 Not Implemented indicating that the
// action isn't done yet.
func (c *Controller) NotImplemented() Result {
	c.Response.Status = http.StatusNotImplemented

	return c.RenderError(&Error{
		Title:       "Not Implemented",
		Description: "Not Implemented",
	})
}
