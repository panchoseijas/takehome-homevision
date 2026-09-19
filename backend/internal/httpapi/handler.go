package httpapi

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"runtime"
	"strconv"
	"time"

	"github.com/panchoseijas/takehome-homevision/backend/internal/vision"
)

type Detector interface {
	Detect(ctx context.Context, data []byte) ([]vision.Box, error)
}

type Config struct {
	MaxUploadBytes int64
	MaxPixels      int
	MaxConcurrent  int
	QueueTimeout   time.Duration
	Logger         *slog.Logger
}

const (
	defaultMaxUploadBytes = 20 << 20
	defaultQueueTimeout   = 5 * time.Second
)

func (c Config) withDefaults() Config {
	if c.MaxUploadBytes <= 0 {
		c.MaxUploadBytes = defaultMaxUploadBytes
	}
	if c.MaxPixels <= 0 {
		c.MaxPixels = vision.DefaultParams().MaxPixels
	}
	if c.MaxConcurrent <= 0 {
		c.MaxConcurrent = runtime.GOMAXPROCS(0)
	}
	if c.QueueTimeout <= 0 {
		c.QueueTimeout = defaultQueueTimeout
	}
	if c.Logger == nil {
		c.Logger = slog.Default()
	}
	return c
}

type server struct {
	detector Detector
	config   Config
	slots    chan struct{}
}

func New(detector Detector, config Config) http.Handler {
	config = config.withDefaults()
	s := &server{
		detector: detector,
		config:   config,
		slots:    make(chan struct{}, config.MaxConcurrent),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /detect", s.handleDetect)
	mux.HandleFunc("GET /healthz", handleHealthz)
	return mux
}

func handleHealthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *server) handleDetect(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, s.config.MaxUploadBytes)

	file, _, err := r.FormFile("image")
	if err != nil {
		status, message := uploadErrorStatus(err)
		writeError(w, status, message)
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		writeError(w, http.StatusBadRequest, "could not read uploaded file")
		return
	}

	if _, err := vision.ValidateImage(data, s.config.MaxPixels); err != nil {
		status, message := detectErrorStatus(err)
		writeError(w, status, message)
		return
	}

	release, ok := s.acquireSlot(r.Context())
	if !ok {
		if r.Context().Err() != nil {
			return // client went away while waiting
		}
		w.Header().Set("Retry-After", strconv.Itoa(int(s.config.QueueTimeout/time.Second)))
		writeError(w, http.StatusServiceUnavailable, "server is busy, retry shortly")
		return
	}
	defer release()

	started := time.Now()
	boxes, err := s.detector.Detect(r.Context(), data)
	if err != nil {
		if r.Context().Err() != nil {
			return
		}
		status, message := detectErrorStatus(err)
		if status == http.StatusInternalServerError {
			s.config.Logger.Error("detection failed", "error", err)
		}
		writeError(w, status, message)
		return
	}
	s.config.Logger.Info(
		"detection complete",
		"boxes", len(boxes),
		"bytes", len(data),
		"duration", time.Since(started),
	)

	writeJSON(w, http.StatusOK, NewDetectResponse(boxes, wantsDebug(r)))
}

func (s *server) acquireSlot(ctx context.Context) (release func(), ok bool) {
	timer := time.NewTimer(s.config.QueueTimeout)
	defer timer.Stop()

	select {
	case s.slots <- struct{}{}:
		return func() { <-s.slots }, true
	case <-ctx.Done():
		return nil, false
	case <-timer.C:
		return nil, false
	}
}

func wantsDebug(r *http.Request) bool {
	switch r.URL.Query().Get("debug") {
	case "1", "true":
		return true
	}
	return false
}

func uploadErrorStatus(err error) (int, string) {
	var maxBytesErr *http.MaxBytesError
	switch {
	case errors.As(err, &maxBytesErr):
		return http.StatusRequestEntityTooLarge, "request body exceeds the upload limit"
	case errors.Is(err, http.ErrNotMultipart):
		return http.StatusBadRequest, "expected a multipart/form-data request"
	case errors.Is(err, http.ErrMissingFile):
		return http.StatusBadRequest, `missing "image" file field`
	default:
		return http.StatusBadRequest, "malformed multipart request"
	}
}

func detectErrorStatus(err error) (int, string) {
	switch {
	case errors.Is(err, vision.ErrUnsupportedFormat):
		return http.StatusUnsupportedMediaType, "uploaded file must be a PNG or JPEG image"
	case errors.Is(err, vision.ErrTooLarge):
		return http.StatusRequestEntityTooLarge, "image dimensions exceed the supported size"
	case errors.Is(err, vision.ErrCorrupt):
		return http.StatusBadRequest, "image could not be decoded"
	default:
		return http.StatusInternalServerError, "detection failed"
	}
}
