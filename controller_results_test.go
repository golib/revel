package revel

import (
	"net/http/httptest"
	"strings"
	"testing"
)

// Test that the render response is as expected.
func TestRenderWithLayout(t *testing.T) {
	fakeTestApp()

	resp := httptest.NewRecorder()

	c := NewController(NewRequest(indexRequest), NewResponse(resp))
	c.SetAction("Hotels", "Index")

	result := Hotels{c}.Index()
	result.Apply(c.Request, c.Response)

	body := resp.Body.String()
	if !strings.Contains(body, "Hotels Layout") {
		t.Errorf("Expect response contains `Hotels Layout`, but got \n%s", body)
		t.FailNow()
	}
	if !strings.Contains(body, "Hotels Index Page") {
		t.Errorf("Expect response contains `Hotels Index Page`, but got \n%s", body)
		t.FailNow()
	}
	if !strings.Contains(body, "San Francisco") {
		t.Errorf("Expect response contains `San Francisco`, but got \n%s", body)
	}
}
