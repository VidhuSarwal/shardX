# Builds both binaries; the image is selected per service via `command`.
FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags='-s -w' -o /out/server ./cmd/server \
 && CGO_ENABLED=0 go build -ldflags='-s -w' -o /out/shardworker ./cmd/shardworker

FROM alpine:3.20
RUN apk add --no-cache ca-certificates curl && adduser -D -u 10001 app
COPY --from=build /out/server /out/shardworker /usr/local/bin/
USER app
ENV PORT=5555 UPLOAD_TEMP_DIR=/data/uploads
EXPOSE 5555
CMD ["server"]
