# Stage 1: Build
FROM golang:1.25 AS builder
WORKDIR /src
RUN apt-get update && apt-get install -y --no-install-recommends libasound2-dev pkg-config && rm -rf /var/lib/apt/lists/*
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN GOOS=linux go build -o lyrike-studio-tui ./cmd/lyrike-studio-tui

# Stage 2: Runtime
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ffmpeg ca-certificates curl python3 && rm -rf /var/lib/apt/lists/*
RUN curl -L -o /usr/local/bin/yt-dlp https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp && chmod +x /usr/local/bin/yt-dlp
COPY --from=builder /src/lyrike-studio-tui /usr/local/bin/lyrike-studio-tui
WORKDIR /data
EXPOSE 8080
ENV LYRIKE_CACHE_DIR=/data/.cache
CMD ["lyrike-studio-tui", "serve"]
