package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewServerRoutesDetect(t *testing.T) {
	server := newServer(":0")

	tests := []struct {
		method     string
		path       string
		wantStatus int
	}{
		{http.MethodPost, "/detect", http.StatusBadRequest},
		{http.MethodGet, "/", http.StatusNotFound},
	}

	for _, tt := range tests {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(tt.method, tt.path, nil)

		server.Handler.ServeHTTP(recorder, request)

		if recorder.Code != tt.wantStatus {
			t.Errorf("%s %s: status = %d, want %d", tt.method, tt.path, recorder.Code, tt.wantStatus)
		}
	}
}
