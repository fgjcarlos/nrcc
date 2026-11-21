package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthzReportsExhaustedRestartBudget(t *testing.T) {
	for _, test := range []struct {
		exhausted bool
		want      int
	}{
		{exhausted: true, want: http.StatusServiceUnavailable},
		{exhausted: false, want: http.StatusOK},
	} {
		rec := httptest.NewRecorder()
		healthz(func() bool { return test.exhausted })(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
		if rec.Code != test.want {
			t.Fatalf("exhausted=%v: status = %d, want %d", test.exhausted, rec.Code, test.want)
		}
	}
}
