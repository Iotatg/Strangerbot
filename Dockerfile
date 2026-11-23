FROM golang:1.27-alpine AS build
WORKDIR /src
RUN apk add --no-cache build-base
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/strangerbot ./cmd/strangerbot

FROM alpine:3.21
RUN apk add --no-cache ca-certificates libstdc++
WORKDIR /app
COPY --from=build /out/strangerbot /app/strangerbot
COPY lib/ /app/lib/
ENV TDLIB_LIBRARY_PATH=/app/lib/libtdjson.so \
    TDLIB_DATABASE_DIR=/app/data/tdlib/db \
    TDLIB_FILES_DIR=/app/data/tdlib/files \
    DB_PATH=/app/data/db.json
VOLUME ["/app/data"]
ENTRYPOINT ["/app/strangerbot"]