# Build with Go and OpenCV, then copy only what the server needs to run.
FROM ghcr.io/hybridgroup/opencv:4.13.0 AS build
# Download the Go version required by go.mod when the base image is older.
ENV GOTOOLCHAIN=auto
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -ldflags="-s -w" -o /server ./cmd/server
# ldd lists the shared libraries, including their transitive dependencies.
RUN mkdir /runtime && ldd /server | awk '/=> \// {print $3} $1 ~ /^\// {print $1}' \
    | xargs -r cp --parents --dereference --target-directory=/runtime

FROM debian:bookworm-slim
COPY --from=build /runtime/lib/ /lib/
COPY --from=build /runtime/lib64/ /lib64/
COPY --from=build /runtime/usr/local/lib/ /usr/local/lib/
COPY --from=build /server /server
RUN ldconfig
CMD ["/server"]
