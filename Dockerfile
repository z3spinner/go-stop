# ── production build ──────────────────────────────────────────────────────────
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o go-stop .

FROM alpine:latest AS production
RUN apk --no-cache add ca-certificates postgresql-client
WORKDIR /app
COPY --from=builder /app/go-stop .
COPY web/ ./web/
EXPOSE 8080
CMD ["./go-stop"]

# ── development (hot-reload via reflex) ───────────────────────────────────────
FROM golang:1.25-alpine AS development
RUN go install github.com/cespare/reflex@latest
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
EXPOSE 8080
# Source is mounted as a volume; reflex watches *.go and rebuilds on change.
CMD ["reflex", "-r", "\\.go$", "-s", "--", "sh", "-c", "go build -o /tmp/go-stop . && /tmp/go-stop"]
