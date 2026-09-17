package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
)

const maxUploadBytes = 20 << 20

func New() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /detect", handleDetect)
	return mux
}

type detectResponse struct {
	Received string `json:"received"`
	Bytes    int    `json:"bytes"`
}

func handleDetect(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)

	file, header, err := r.FormFile("image")
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
	if !strings.HasPrefix(http.DetectContentType(data), "image/") {
		writeError(w, http.StatusUnsupportedMediaType, "uploaded file is not an image")
		return
	}

	writeJSON(w, http.StatusOK, detectResponse{Received: header.Filename, Bytes: len(data)})
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

type errorResponse struct {
	Error string `json:"error"`
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
