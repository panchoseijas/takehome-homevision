package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewServerHasNoRoutes(t *testing.T) {
	server := newServer(":0")

	for _, path := range []string{"/", "/detect", "/healthz"} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, path, nil)

		server.Handler.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusNotFound {
			t.Errorf("GET %s: status = %d, want %d", path, recorder.Code, http.StatusNotFound)
		}
	}
}
