FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /gitant ./cmd/gitant/
RUN CGO_ENABLED=0 go build -o /git-remote-gitant ./cmd/git-remote-gitant/

FROM alpine:3.20
RUN apk add --no-cache git ca-certificates
COPY --from=builder /gitant /usr/local/bin/
COPY --from=builder /git-remote-gitant /usr/local/bin/
EXPOSE 7777
VOLUME ["/root/.gitant"]
ENTRYPOINT ["gitant"]
CMD ["serve"]
