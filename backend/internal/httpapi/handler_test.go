package httpapi

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

var minimalPNG = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR")

func multipartUpload(t *testing.T, field, filename string, content []byte) *http.Request {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile(field, filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodPost, "/detect", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	return request
}

func TestDetectAcceptsImageUpload(t *testing.T) {
	recorder := httptest.NewRecorder()
	New().ServeHTTP(recorder, multipartUpload(t, "image", "form.png", minimalPNG))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", recorder.Code, http.StatusOK, recorder.Body)
	}
	var response detectResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Received != "form.png" || response.Bytes != len(minimalPNG) {
		t.Errorf("response = %+v, want received=form.png bytes=%d", response, len(minimalPNG))
	}
}

func TestDetectRejectsBadUploads(t *testing.T) {
	tests := []struct {
		name       string
		request    *http.Request
		wantStatus int
	}{
		{
			name:       "not multipart",
			request:    httptest.NewRequest(http.MethodPost, "/detect", strings.NewReader("{}")),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "wrong field name",
			request:    multipartUpload(t, "file", "form.png", minimalPNG),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "not an image",
			request:    multipartUpload(t, "image", "notes.txt", []byte("plain text")),
			wantStatus: http.StatusUnsupportedMediaType,
		},
		{
			name:       "wrong method",
			request:    httptest.NewRequest(http.MethodGet, "/detect", nil),
			wantStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			New().ServeHTTP(recorder, tt.request)

			if recorder.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d; body: %s", recorder.Code, tt.wantStatus, recorder.Body)
			}
		})
	}
}
