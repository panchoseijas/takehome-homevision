# GoCV's base image includes Go and the matching OpenCV libraries.
FROM ghcr.io/hybridgroup/opencv:4.13.0
# Download the Go version required by go.mod when the base image is older.
ENV GOTOOLCHAIN=auto
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /server ./cmd/server
CMD ["/server"]
