package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/panchoseijas/takehome-homevision/backend/internal/vision"
)

// fakeDetector returns canned results and records what it received.
type fakeDetector struct {
	boxes   []vision.Box
	err     error
	block   chan struct{} // when set, Detect waits until closed
	mu      sync.Mutex
	inputs  [][]byte
	calls   int
	lastCtx context.Context
}

func (f *fakeDetector) Detect(ctx context.Context, data []byte) ([]vision.Box, error) {
	f.mu.Lock()
	f.calls++
	f.inputs = append(f.inputs, data)
	f.lastCtx = ctx
	f.mu.Unlock()
	if f.block != nil {
		select {
		case <-f.block:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return f.boxes, f.err
}

func newHandler(t *testing.T, detector Detector, config Config) http.Handler {
	t.Helper()
	config.Logger = slog.New(slog.DiscardHandler)
	return New(detector, config)
}

func encodePNG(t *testing.T, width, height int) []byte {
	t.Helper()
	img := image.NewGray(image.Rect(0, 0, width, height))
	for i := range img.Pix {
		img.Pix[i] = 255
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func encodeJPEG(t *testing.T, width, height int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		for x := range width {
			img.Set(x, y, color.White)
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func multipartUpload(t *testing.T, target, field, filename string, content []byte) *http.Request {
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

	request := httptest.NewRequest(http.MethodPost, target, &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	return request
}

func decodeBody(t *testing.T, recorder *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	if got := recorder.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", got)
	}
	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body %q: %v", recorder.Body, err)
	}
	return body
}

var sampleDebug = vision.Debug{
	FillRatio:    0.25,
	InkPixels:    100,
	InteriorArea: 400,
	BorderPx:     [4]int{2, 2, 2, 2},
}

var sampleBoxes = []vision.Box{
	{X1: 10, Y1: 20, X2: 50, Y2: 60, Checked: true, Debug: sampleDebug},
	{X1: 100, Y1: 20, X2: 140, Y2: 60, Checked: false},
}

func TestDetectReturnsContractShape(t *testing.T) {
	detector := &fakeDetector{boxes: sampleBoxes}
	handler := newHandler(t, detector, Config{})
	upload := encodePNG(t, 64, 64)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, multipartUpload(t, "/detect", "image", "form.png", upload))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", recorder.Code, recorder.Body)
	}
	want := `{"boxes":[{"bbox":[10,20,50,60],"is_checked":true},{"bbox":[100,20,140,60],"is_checked":false}]}`
	if got := strings.TrimSpace(recorder.Body.String()); got != want {
		t.Errorf("body = %s\nwant   %s", got, want)
	}
	if len(detector.inputs) != 1 || !bytes.Equal(detector.inputs[0], upload) {
		t.Error("detector did not receive the uploaded bytes")
	}
}

func TestDetectEmptyResultIsEmptyArray(t *testing.T) {
	handler := newHandler(t, &fakeDetector{}, Config{})

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, multipartUpload(t, "/detect", "image", "blank.jpg", encodeJPEG(t, 32, 32)))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", recorder.Code, recorder.Body)
	}
	if got := strings.TrimSpace(recorder.Body.String()); got != `{"boxes":[]}` {
		t.Errorf("body = %s, want {\"boxes\":[]}", got)
	}
}

func TestDetectDebugVariant(t *testing.T) {
	handler := newHandler(t, &fakeDetector{boxes: sampleBoxes}, Config{})

	for _, target := range []string{"/detect?debug=1", "/detect?debug=true"} {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, multipartUpload(t, target, "image", "form.png", encodePNG(t, 64, 64)))

		if recorder.Code != http.StatusOK {
			t.Fatalf("%s: status = %d, body %s", target, recorder.Code, recorder.Body)
		}
		var response DetectResponse
		if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		if len(response.Boxes) != 2 || response.Boxes[0].Debug == nil || response.Boxes[1].Debug == nil {
			t.Fatalf("%s: every box should carry debug: %s", target, recorder.Body)
		}
		want := DebugResponse{
			FillRatio:    sampleDebug.FillRatio,
			InkPixels:    sampleDebug.InkPixels,
			InteriorArea: sampleDebug.InteriorArea,
			BorderPx:     sampleDebug.BorderPx,
		}
		if got := *response.Boxes[0].Debug; got != want {
			t.Errorf("%s: debug = %+v, want %+v", target, got, want)
		}
	}

	// Any other value keeps the default shape.
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, multipartUpload(t, "/detect?debug=0", "image", "form.png", encodePNG(t, 64, 64)))
	if strings.Contains(recorder.Body.String(), "debug") {
		t.Errorf("debug=0 should not include diagnostics: %s", recorder.Body)
	}
}

func TestDetectRejectsBadUploads(t *testing.T) {
	tests := []struct {
		name       string
		request    *http.Request
		wantStatus int
		wantError  string
	}{
		{
			name:       "not multipart",
			request:    httptest.NewRequest(http.MethodPost, "/detect", strings.NewReader("{}")),
			wantStatus: http.StatusBadRequest,
			wantError:  "expected a multipart/form-data request",
		},
		{
			name:       "wrong field name",
			request:    multipartUpload(t, "/detect", "file", "form.png", encodePNG(t, 8, 8)),
			wantStatus: http.StatusBadRequest,
			wantError:  `missing "image" file field`,
		},
		{
			name:       "not an image",
			request:    multipartUpload(t, "/detect", "image", "notes.txt", []byte("plain text")),
			wantStatus: http.StatusUnsupportedMediaType,
			wantError:  "uploaded file must be a PNG or JPEG image",
		},
		{
			name:       "gif",
			request:    multipartUpload(t, "/detect", "image", "anim.gif", []byte("GIF89a\x01\x00\x01\x00\x00\x00\x00;")),
			wantStatus: http.StatusUnsupportedMediaType,
			wantError:  "uploaded file must be a PNG or JPEG image",
		},
		{
			name:       "truncated png",
			request:    multipartUpload(t, "/detect", "image", "cut.png", encodePNG(t, 8, 8)[:20]),
			wantStatus: http.StatusBadRequest,
			wantError:  "image could not be decoded",
		},
		{
			name:       "oversized dimensions",
			request:    multipartUpload(t, "/detect", "image", "huge.png", encodePNG(t, 200, 200)),
			wantStatus: http.StatusRequestEntityTooLarge,
			wantError:  "image dimensions exceed the supported size",
		},
		{
			name:       "oversized body",
			request:    multipartUpload(t, "/detect", "image", "big.png", bytes.Repeat([]byte{0}, 5000)),
			wantStatus: http.StatusRequestEntityTooLarge,
			wantError:  "request body exceeds the upload limit",
		},
		{
			name:       "wrong method",
			request:    httptest.NewRequest(http.MethodGet, "/detect", nil),
			wantStatus: http.StatusMethodNotAllowed,
		},
	}

	detector := &fakeDetector{}
	handler := newHandler(t, detector, Config{MaxPixels: 10_000, MaxUploadBytes: 4096})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, tt.request)

			if recorder.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d; body: %s", recorder.Code, tt.wantStatus, recorder.Body)
			}
			if tt.wantError != "" {
				if got := decodeBody(t, recorder)["error"]; got != tt.wantError {
					t.Errorf("error = %q, want %q", got, tt.wantError)
				}
			}
		})
	}
	if detector.calls != 0 {
		t.Errorf("detector ran %d times for invalid uploads", detector.calls)
	}
}

func TestDetectMapsDetectorErrors(t *testing.T) {
	tests := []struct {
		err        error
		wantStatus int
	}{
		{vision.ErrCorrupt, http.StatusBadRequest},
		{vision.ErrUnsupportedFormat, http.StatusUnsupportedMediaType},
		{vision.ErrTooLarge, http.StatusRequestEntityTooLarge},
		{errors.New("opencv exploded"), http.StatusInternalServerError},
	}
	for _, tt := range tests {
		handler := newHandler(t, &fakeDetector{err: tt.err}, Config{})
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, multipartUpload(t, "/detect", "image", "form.png", encodePNG(t, 8, 8)))

		if recorder.Code != tt.wantStatus {
			t.Errorf("%v: status = %d, want %d", tt.err, recorder.Code, tt.wantStatus)
		}
		if body := recorder.Body.String(); strings.Contains(body, "exploded") {
			t.Errorf("internal error details leaked: %s", body)
		}
	}
}

func TestDetectReturnsBusyWhenSlotsAreTaken(t *testing.T) {
	detector := &fakeDetector{block: make(chan struct{})}
	handler := newHandler(t, detector, Config{MaxConcurrent: 1, QueueTimeout: 50 * time.Millisecond})
	upload := encodePNG(t, 8, 8)

	firstDone := make(chan *httptest.ResponseRecorder)
	go func() {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, multipartUpload(t, "/detect", "image", "a.png", upload))
		firstDone <- recorder
	}()
	waitForCalls(t, detector, 1)

	second := httptest.NewRecorder()
	handler.ServeHTTP(second, multipartUpload(t, "/detect", "image", "b.png", upload))
	if second.Code != http.StatusServiceUnavailable {
		t.Errorf("second status = %d, want 503; body %s", second.Code, second.Body)
	}
	if second.Header().Get("Retry-After") == "" {
		t.Error("503 should carry Retry-After")
	}

	close(detector.block)
	if first := <-firstDone; first.Code != http.StatusOK {
		t.Errorf("first status = %d, want 200", first.Code)
	}

	// The slot is released, so a new request goes through.
	third := httptest.NewRecorder()
	handler.ServeHTTP(third, multipartUpload(t, "/detect", "image", "c.png", upload))
	if third.Code != http.StatusOK {
		t.Errorf("third status = %d, want 200", third.Code)
	}
}

func waitForCalls(t *testing.T, detector *fakeDetector, want int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		detector.mu.Lock()
		calls := detector.calls
		detector.mu.Unlock()
		if calls >= want {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("detector was not called %d times", want)
}

func TestHealthz(t *testing.T) {
	handler := newHandler(t, &fakeDetector{}, Config{})

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d", recorder.Code)
	}
	if got := decodeBody(t, recorder)["status"]; got != "ok" {
		t.Errorf("status field = %v, want ok", got)
	}
}
