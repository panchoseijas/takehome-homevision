package main

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewServerRoutes(t *testing.T) {
	server := newServer(":0", 1, slog.New(slog.DiscardHandler))

	tests := []struct {
		method     string
		path       string
		wantStatus int
	}{
		{http.MethodPost, "/detect", http.StatusBadRequest},
		{http.MethodGet, "/healthz", http.StatusOK},
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
