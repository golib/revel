package revel

import (
	"net/http"
	"strings"
	"testing"
)

func TestHttpMethodOverrideFilter(t *testing.T) {
	req, _ := http.NewRequest("POST", "/hotels/3", strings.NewReader("_method=put"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")

	ctrl := Controller{
		Request: NewRequest(req),
	}

	if HttpMethodOverrideFilter(&ctrl, NilChain); ctrl.Request.Request.Method != "PUT" {
		t.Errorf("Expected to override current method 'PUT' in route, found '%s' instead", ctrl.Request.Request.Method)
	}
}
