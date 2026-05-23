FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /gitant ./cmd/gitant/
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /git-remote-gitant ./cmd/git-remote-gitant/

FROM alpine:3.20
RUN apk add --no-cache git ca-certificates && \
    adduser -D -u 1000 gitant
COPY --from=builder /gitant /usr/local/bin/
COPY --from=builder /git-remote-gitant /usr/local/bin/
USER gitant
WORKDIR /home/gitant
EXPOSE 7777
VOLUME ["/home/gitant/.gitant"]
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget --spider -q http://localhost:7777/health || exit 1
ENTRYPOINT ["gitant"]
CMD ["serve"]
