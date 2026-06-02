# ── frontend build ──
FROM node:22-alpine AS frontend
WORKDIR /app
COPY frontend/package.json frontend/package-lock.json ./frontend/
RUN cd frontend && npm ci
COPY frontend ./frontend
RUN cd frontend && npm run build   # outputs to /app/web/build

# ── go build ──
FROM golang:1.25-alpine AS builder
ARG GIT_SHA=dev
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN printf 'package version\n\nvar Build = "%s"\n' "${GIT_SHA}" > internal/version/build.go
RUN CGO_ENABLED=0 go build -o go-stop . && CGO_ENABLED=0 go build -o migratedb ./cmd/migratedb

# ── production ──
FROM alpine:latest AS production
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/go-stop .
COPY --from=builder /app/migratedb .
COPY --from=frontend /app/web/build ./web/build
EXPOSE 8080
CMD ["./go-stop"]

# ── development (Go hot-reload; frontend runs via vite separately) ──
FROM golang:1.25-alpine AS development
RUN go install github.com/cespare/reflex@latest
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
EXPOSE 8080
CMD ["reflex", "-r", "\\.go$", "-s", "--", "sh", "-c", "go build -o /tmp/go-stop . && /tmp/go-stop"]
