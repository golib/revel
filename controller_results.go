package revel

import (
	"bytes"
	"errors"
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

// Render a template and capture the result
func (c *Controller) CaptureTemplate(name string) (html template.HTML, err error) {
	// Handle panics when rendering templates.
	defer func() {
		if panicErr := recover(); panicErr != nil {
			err = errors.New(fmt.Sprintf("Template Execution Panic in %s : %s\n", name, panicErr))
			return
		}
	}()

	// always using lowercase
	name = strings.ToLower(name)

	// Get the Template.
	goTemplate, err := MainTemplateLoader.Template(name)
	if err != nil {
		return "", errors.New(fmt.Sprintf("Failed loading template %s : %s\n", name, err.Error()))
	}

	buf := bytes.NewBuffer([]byte{})

	// Internal help template variable for some purposes?
	c.RenderArgs["RevelTemplateCaptureMode"] = true

	err = goTemplate.Render(buf, c.RenderArgs)
	if err != nil {
		return "", errors.New(fmt.Sprintf("Failed rendering template %s : %s\n", name, err.Error()))
	}

	return template.HTML(buf.String()), nil
}

// Render a template corresponding to the calling Controller method.
// Arguments will be added to c.RenderArgs prior to rendering the template.
// They are keyed on their local identifier.
//
// For example:
//
//     func (c Users) ShowUser(id int) revel.Result {
//     	 user := loadUser(id)
//     	 return c.Render(user)
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

	// apply all yielded blocks
	yield2blocks, err := MainTemplateLoader.Yield2Blocks(name)
	if err == nil {
		for yieldName, blockName := range yield2blocks {
			blockHtml, blockErr := c.CaptureTemplate(blockName)

			if blockErr != nil {
				goTemplate, err := MainTemplateLoader.Template(name)
				if err != nil {
					return c.RenderError(err)
				}

				_, line, description := parseTemplateError(blockErr)

				return c.RenderError(&Error{
					Title:       "Template Execution Error",
					Line:        line,
					Path:        name,
					Description: description,
					SourceLines: goTemplate.Content(),
				})
			}

			c.RenderArgs[yieldName] = blockHtml
		}
	}

	// layout?
	templateName := name
	if layouter, ok := c.AppController.(ControllerLayouter); ok {
		layouts := layouter.Layout()

		// is there an action layout defined?
		if layout, ok := layouts[c.MethodName]; ok {
			templateName = layout

			// is there an method:action layout defined?
		} else if layout, ok := layouts[c.Request.Method+":"+c.MethodName]; ok {
			templateName = layout

			// is there an wildcard layout defined?
		} else if layout, ok := layouts["*"]; ok {
			templateName = layout
		}
	}

	// Get the Template.
	goTemplate, err := MainTemplateLoader.Template(templateName)
	if err != nil {
		return c.RenderError(err)
	}

	return &RenderTemplateResult{
		Template:   goTemplate,
		RenderArgs: c.RenderArgs,
	}
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
