package revel

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Test that the render response is as expected.
func TestRenderTemplateResult(t *testing.T) {
	fakeTestApp()

	resp := httptest.NewRecorder()

	c := NewController(NewRequest(showRequest), NewResponse(resp))
	c.SetAction("Hotels", "Show")

	result := Hotels{c}.Show(3)
	result.Apply(c.Request, c.Response)

	if resp.Code != http.StatusOK {
		t.Errorf("Expect respond status `%d` but got `%d`", http.StatusOK, resp.Code)
	}
	if !strings.Contains(resp.HeaderMap.Get("Content-Type"), "text/html") {
		t.Errorf("Expect respond content type `text/html` but got `%s`", resp.HeaderMap.Get("Content-Type"))
	}
	if resp.HeaderMap.Get("Content-Length") != fmt.Sprintf("%d", len(resp.Body.String())) {
		t.Errorf("Expect respond content length `%d` but got `%s`", len(resp.Body.String()), resp.HeaderMap.Get("Content-Length"))
	}
	if !strings.Contains(resp.Body.String(), "Hotels Show Page") {
		t.Errorf("Expect respond with `Hotels Show Page` but got `%s`", resp.Body.String())
	}
}

func TestRenderTemplateResultWithChunked(t *testing.T) {
	fakeTestApp()

	resp := httptest.NewRecorder()

	c := NewController(NewRequest(showRequest), NewResponse(resp))
	c.SetAction("Hotels", "Show")

	Config.SetOption("results.chunked", "true")

	result := Hotels{c}.Show(3)
	result.Apply(c.Request, c.Response)

	if resp.Code != http.StatusOK {
		t.Errorf("Expect respond status `%d` but got `%d`", http.StatusOK, resp.Code)
	}
	if !strings.Contains(resp.HeaderMap.Get("Content-Type"), "text/html") {
		t.Errorf("Expect respond content type `text/html` but got `%s`", resp.HeaderMap.Get("Content-Type"))
	}
	if resp.HeaderMap.Get("Content-Length") != "" {
		t.Errorf("Expect respond content length `` but got `%s`", resp.HeaderMap.Get("Content-Length"))
	}
	if !strings.Contains(resp.Body.String(), "Hotels Show Page") {
		t.Errorf("Expect respond with `Hotels Show Page` but got `%s`", resp.Body.String())
	}
}

func TestRenderTemplateResultWithError(t *testing.T) {
	fakeTestApp()

	resp := httptest.NewRecorder()

	c := NewController(NewRequest(errorRequest), NewResponse(resp))
	c.SetAction("Hotels", "Error")

	result := Hotels{c}.Error()
	result.Apply(c.Request, c.Response)

	if resp.Code != http.StatusInternalServerError {
		t.Errorf("Expect respond status `%d` but got `%d`", http.StatusInternalServerError, resp.Code)
	}
	if !strings.Contains(resp.HeaderMap.Get("Content-Type"), "text/html") {
		t.Errorf("Expect respond content type `text/html` but got `%s`", resp.HeaderMap.Get("Content-Type"))
	}
	if resp.HeaderMap.Get("Content-Length") != "" {
		t.Errorf("Expect respond content length `` but got `%s`", resp.HeaderMap.Get("Content-Length"))
	}
	if !strings.Contains(resp.Body.String(), "Oops, an error occured") {
		t.Errorf("Expect respond with `Oops, an error occured` but got `%s`", resp.Body.String())
	}
}

func TestRenderTemplateResultWithPanic(t *testing.T) {
	fakeTestApp()

	resp := httptest.NewRecorder()

	c := NewController(NewRequest(showRequest), NewResponse(resp))
	c.SetAction("Hotels", "Panic")

	result := Hotels{c}.Panic()
	result.Apply(c.Request, c.Response)

	if resp.Code != http.StatusInternalServerError {
		t.Errorf("Expect respond status `%d` but got `%d`", http.StatusInternalServerError, resp.Code)
	}
	if !strings.Contains(resp.HeaderMap.Get("Content-Type"), "text/plain") {
		t.Errorf("Expect respond content type `text/plain` but got `%s`", resp.HeaderMap.Get("Content-Type"))
	}
	if resp.HeaderMap.Get("Content-Length") != "" {
		t.Errorf("Expect respond content length `` but got `%s`", resp.HeaderMap.Get("Content-Length"))
	}
	if !strings.Contains(resp.Body.String(), "Template Execution Panic") {
		t.Errorf("Expect respond with `Template Execution Panic` but got `%s`", resp.Body.String())
	}
}

func TestRenderTextResult(t *testing.T) {
	resp := httptest.NewRecorder()

	s := "Plain text result"

	result := RenderTextResult{text: s}
	result.Apply(NewRequest(showRequest), NewResponse(resp))

	if resp.Code != http.StatusOK {
		t.Errorf("Expect respond status `%d` but got `%d`", http.StatusOK, resp.Code)
	}
	if !strings.Contains(resp.HeaderMap.Get("Content-Type"), "text/plain") {
		t.Errorf("Expect respond content type `text/plain` but got `%s`", resp.HeaderMap.Get("Content-Type"))
	}
	if !strings.Contains(resp.Body.String(), s) {
		t.Errorf("Expect respond content `"+s+"` but got `%s`", resp.Body.String())
	}
}

func TestRenderHtmlResult(t *testing.T) {
	resp := httptest.NewRecorder()

	s := "<h1>Revel</h1>"

	result := RenderHtmlResult{html: s}
	result.Apply(NewRequest(showRequest), NewResponse(resp))

	if resp.Code != http.StatusOK {
		t.Errorf("Expect respond status `%d` but got `%d`", http.StatusOK, resp.Code)
	}
	if !strings.Contains(resp.HeaderMap.Get("Content-Type"), "text/html") {
		t.Errorf("Expect respond content type `text/html` but got `%s`", resp.HeaderMap.Get("Content-Type"))
	}
	if !strings.Contains(resp.Body.String(), s) {
		t.Errorf("Expect respond content `"+s+"` but got `%s`", resp.Body.String())
	}
}

func TestRenderJsonResult(t *testing.T) {
	resp := httptest.NewRecorder()

	json := map[string]interface{}{
		"success": true,
		"data":    []string{"revel"},
		"message": "Json result",
	}

	result := RenderJsonResult{json: json}
	result.Apply(NewRequest(showRequest), NewResponse(resp))

	if resp.Code != http.StatusOK {
		t.Errorf("Expect respond status `%d` but got `%d`", http.StatusOK, resp.Code)
	}
	if !strings.Contains(resp.HeaderMap.Get("Content-Type"), "application/json") {
		t.Errorf("Expect respond content type `application/json` but got `%s`", resp.HeaderMap.Get("Content-Type"))
	}
	if !strings.Contains(resp.Body.String(), "Json result") {
		t.Errorf("Expect respond content `Json result` but got `%s`", resp.Body.String())
	}
}

func TestRenderJsonPResult(t *testing.T) {
	resp := httptest.NewRecorder()

	json := map[string]interface{}{
		"success": true,
		"data":    []string{"revel"},
		"message": "Json result",
	}

	result := RenderJsonResult{json: json, callback: "revel_callback"}
	result.Apply(NewRequest(showRequest), NewResponse(resp))

	if resp.Code != http.StatusOK {
		t.Errorf("Expect respond status `%d` but got `%d`", http.StatusOK, resp.Code)
	}
	if !strings.Contains(resp.HeaderMap.Get("Content-Type"), "application/javascript") {
		t.Errorf("Expect respond content type `application/javascript` but got `%s`", resp.HeaderMap.Get("Content-Type"))
	}
	if !strings.Contains(resp.Body.String(), "revel_callback") {
		t.Errorf("Expect respond content `revel_callback` but got `%s`", resp.Body.String())
	}
}

func TestRenderXmlResult(t *testing.T) {
	resp := httptest.NewRecorder()

	type xmlResult struct {
		Success bool
		Data    interface{}
		Message string
	}

	xml := xmlResult{
		Success: true,
		Data:    []string{"revel"},
		Message: "Json result",
	}

	result := RenderXmlResult{xml: xml}
	result.Apply(NewRequest(showRequest), NewResponse(resp))

	if resp.Code != http.StatusOK {
		t.Errorf("Expect respond status `%d` but got `%d`", http.StatusOK, resp.Code)
	}
	if !strings.Contains(resp.HeaderMap.Get("Content-Type"), "application/xml") {
		t.Errorf("Expect respond content type `application/xml` but got `%s`", resp.HeaderMap.Get("Content-Type"))
	}
	if !strings.Contains(resp.Body.String(), "Json result") {
		t.Errorf("Expect respond content `Json result` but got `%s`", resp.Body.String())
	}
}

func TestPlaintextErrorResult(t *testing.T) {
	resp := httptest.NewRecorder()
	err := errors.New("Plain text error result")

	result := PlaintextErrorResult{Error: err}
	result.Apply(NewRequest(errorRequest), NewResponse(resp))

	if resp.Code != http.StatusInternalServerError {
		t.Errorf("Expect respond status `%d` but got `%d`", http.StatusInternalServerError, resp.Code)
	}
	if !strings.Contains(resp.HeaderMap.Get("Content-Type"), "text/plain") {
		t.Errorf("Expect respond content type `text/plain` but got `%s`", resp.HeaderMap.Get("Content-Type"))
	}
	if !strings.Contains(resp.Body.String(), "Plain text error result") {
		t.Errorf("Expect respond content `Plain text error result` but got `%s`", resp.Body.String())
	}
}

func TestErrorResult(t *testing.T) {
	resp := httptest.NewRecorder()
	err := errors.New("Plain text error result")

	result := ErrorResult{Error: err}
	result.Apply(NewRequest(errorRequest), NewResponse(resp))

	if resp.Code != http.StatusInternalServerError {
		t.Errorf("Expect respond status `%d` but got `%d`", http.StatusInternalServerError, resp.Code)
	}
	if !strings.Contains(resp.HeaderMap.Get("Content-Type"), "text/html") {
		t.Errorf("Expect respond content type `text/html` but got `%s`", resp.HeaderMap.Get("Content-Type"))
	}
	if !strings.Contains(resp.Body.String(), "Oops, an error occured") {
		t.Errorf("Expect respond content `Oops, an error occured` but got `%s`", resp.Body.String())
	}
}

func TestErrorResultWithFormat(t *testing.T) {
	req := NewRequest(errorRequest)
	resp := httptest.NewRecorder()
	err := errors.New("Json error result")

	req.Format = "json"

	result := ErrorResult{Error: err}
	result.Apply(req, NewResponse(resp))

	if resp.Code != http.StatusInternalServerError {
		t.Errorf("Expect respond status `%d` but got `%d`", http.StatusInternalServerError, resp.Code)
	}
	if !strings.Contains(resp.HeaderMap.Get("Content-Type"), "application/json") {
		t.Errorf("Expect respond content type `application/json` but got `%s`", resp.HeaderMap.Get("Content-Type"))
	}
	if !strings.Contains(resp.Body.String(), "Json error result") {
		t.Errorf("Expect respond content `Json error result` but got `%s`", resp.Body.String())
	}
}

func TestErrorResultWithStatus(t *testing.T) {
	req := NewRequest(errorRequest)
	resp := httptest.NewRecorder()
	rresp := NewResponse(resp)
	err := errors.New("Json error result")

	rresp.Status = 512

	result := ErrorResult{Error: err}
	result.Apply(req, rresp)

	if resp.Code != 512 {
		t.Errorf("Expect respond status `%d` but got `%d`", 512, resp.Code)
	}
	if !strings.Contains(resp.HeaderMap.Get("Content-Type"), "text/plain") {
		t.Errorf("Expect respond content type `text/plain` but got `%s`", resp.HeaderMap.Get("Content-Type"))
	}
	if !strings.Contains(resp.Body.String(), "Template errors/512.html not found") {
		t.Errorf("Expect respond content `Template errors/512.html not found` but got `%s`", resp.Body.String())
	}
}

func TestErrorResultWithApp(t *testing.T) {
	resp := httptest.NewRecorder()
	rresp := NewResponse(resp)
	err := errors.New("Plain text error result")

	rresp.Status = http.StatusForbidden

	result := ErrorResult{Error: err}
	result.Apply(NewRequest(errorRequest), rresp)

	if resp.Code != http.StatusForbidden {
		t.Errorf("Expect respond status `%d` but got `%d`", http.StatusForbidden, resp.Code)
	}
	if !strings.Contains(resp.HeaderMap.Get("Content-Type"), "text/html") {
		t.Errorf("Expect respond content type `text/html` but got `%s`", resp.HeaderMap.Get("Content-Type"))
	}
	if !strings.Contains(resp.Body.String(), "Hotels 403 Error Page") {
		t.Errorf("Expect respond content `Hotels 403 Error Page` but got `%s`", resp.Body.String())
	}
}

func TestRedirectToUrlResult(t *testing.T) {
	resp := httptest.NewRecorder()

	url := "/hotels/1"

	result := RedirectToUrlResult{url: url}
	result.Apply(NewRequest(showRequest), NewResponse(resp))

	if resp.Code != http.StatusFound {
		t.Errorf("Expect respond status `%d` but got `%d`", http.StatusFound, resp.Code)
	}
	if !strings.Contains(resp.Header().Get("Location"), url) {
		t.Errorf("Expect respond location `"+url+"` but got `%s`", resp.Header().Get("Location"))
	}
}

func TestRedirectToActionResult(t *testing.T) {
	resp := httptest.NewRecorder()

	c := NewController(NewRequest(showRequest), NewResponse(resp))
	c.SetAction("Hotels", "Show")

	result := RedirectToActionResult{val: Hotels.Show, args: map[string]string{"id": "1"}}
	result.Apply(c.Request, c.Response)

	if resp.Code != http.StatusFound {
		t.Errorf("Expect respond status `%d` but got `%d`", http.StatusFound, resp.Code)
	}
	if !strings.Contains(resp.Header().Get("Location"), "/hotels/1") {
		t.Errorf("Expect respond location `/hotels/1` but got `%s`", resp.Header().Get("Location"))
	}
}

func BenchmarkRenderChunked(b *testing.B) {
	fakeTestApp()

	resp := httptest.NewRecorder()
	resp.Body = nil

	c := NewController(NewRequest(showRequest), NewResponse(resp))
	c.SetAction("Hotels", "Show")

	Config.SetOption("results.chunked", "true")

	b.ResetTimer()

	hotels := Hotels{c}
	for i := 0; i < b.N; i++ {
		hotels.Show(3).Apply(c.Request, c.Response)
	}
}

func BenchmarkRenderNotChunked(b *testing.B) {
	fakeTestApp()

	resp := httptest.NewRecorder()
	resp.Body = nil

	c := NewController(NewRequest(showRequest), NewResponse(resp))
	c.SetAction("Hotels", "Show")

	Config.SetOption("results.chunked", "false")

	b.ResetTimer()

	hotels := Hotels{c}
	for i := 0; i < b.N; i++ {
		hotels.Show(3).Apply(c.Request, c.Response)
	}
}
