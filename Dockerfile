# ── production build ──────────────────────────────────────────────────────────
FROM golang:1.25-alpine AS builder
ARG GIT_SHA=dev
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Inject the build SHA into version/build.go before compiling
RUN printf 'package version\n\nvar Build = "%s"\n' "${GIT_SHA}" > internal/version/build.go
RUN CGO_ENABLED=0 go build -o go-stop . && CGO_ENABLED=0 go build -o migratedb ./cmd/migratedb

FROM alpine:latest AS production
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/go-stop .
COPY --from=builder /app/migratedb .
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
