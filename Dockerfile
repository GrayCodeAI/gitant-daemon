# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-s -w" -o /gitant ./cmd/gitant/
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-s -w" -o /git-remote-gitant ./cmd/git-remote-gitant/

# Runtime stage
FROM alpine:3.19

RUN apk --no-cache add ca-certificates curl

WORKDIR /app

COPY --from=builder /gitant /usr/local/bin/gitant
COPY --from=builder /git-remote-gitant /usr/local/bin/git-remote-gitant

RUN addgroup -g 1000 -S gitant && \
    adduser -u 1000 -S gitant -G gitant

RUN mkdir -p /data && chown -R gitant:gitant /data

USER gitant

EXPOSE 7777

HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
  CMD curl -f http://localhost:7777/health || exit 1

ENTRYPOINT ["gitant"]
CMD ["serve"]
