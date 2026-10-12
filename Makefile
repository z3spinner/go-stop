# ── Test stacks ──────────────────────────────────────────────────────────────
# Integration and e2e each run in their own isolated docker compose project with
# NO published host ports, so they never touch the devstack DB/ports and can run
# in parallel with the devstack and each other. The DB is tmpfs (fresh per run);
# the module/node caches persist across runs (so `down` keeps named volumes).
ITEST := docker compose -p gostop-itest -f docker-compose.itest.yml
E2E   := docker compose -p gostop-e2e   -f docker-compose.e2e.yml

# Integration tests in an isolated stack (own Postgres). Leaves the devstack alone.
test-integration:
	@$(ITEST) up --build --abort-on-container-exit --exit-code-from tests; \
	  code=$$?; \
	  $(ITEST) down --remove-orphans >/dev/null 2>&1; \
	  exit $$code

# Back-compat: `make test` runs the integration stack.
test: test-integration

# Pure unit tests (no DB), run directly on the host.
test-unit:
	go test ./internal/usecase/...

# Pinned golangci-lint version. Bump here and in .golangci.yml's header together.
GOLANGCI_LINT_VERSION := v1.64.8

# Install the pinned linter into $(go env GOPATH)/bin.
lint-install:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

# Lint the Go backend (config in .golangci.yml). Run `make lint-install` first if missing.
lint:
	golangci-lint run ./...

# Auto-fix formatting (gofmt + import grouping) for hand-written Go (skips generated
# sqlc queries and swaggo docs).
fmt:
	gofmt -w $(shell git ls-files '*.go' | grep -vE '/sqlc/queries/|^docs/')
	golangci-lint run --fix --enable-only=goimports ./... || true

sqlc:
	sqlc generate

# Seed the dev DB with realistic rides + alerts via the running app's API.
# Useful after `make test`, which truncates the dev database.
# Override the target with BASE_URL=... (default http://localhost:8080).
seed:
	./scripts/seed.sh

build-web:
	npm ci --prefix frontend && npm run build --prefix frontend

dev:
	@echo "Go :8080 + Vite :5173 (proxying /api). Ctrl-C stops both."
	@( go run . & echo $$! > /tmp/gostop-go.pid ; npm run dev --prefix frontend ; kill `cat /tmp/gostop-go.pid` )

# End-to-end tests in an isolated stack: the production app image (self-contained
# SPA + API) + its own Postgres + a Playwright runner — all on a private network.
test-e2e:
	@$(E2E) up --build --abort-on-container-exit --exit-code-from tests; \
	  code=$$?; \
	  $(E2E) down --remove-orphans >/dev/null 2>&1; \
	  exit $$code

# Everything: unit (host), then the integration and e2e stacks in parallel.
test-all: test-unit
	@$(MAKE) -j2 test-integration test-e2e

swagger-install:
	go install github.com/swaggo/swag/cmd/swag@latest

# --propertyStrategy pascalcase: tag-less domain structs (domain.Ride/Request) then
# serialize as PascalCase to match the real Gin wire format; json-tagged DTOs are
# unaffected because swag reads their json tags first.
swagger:
	swag init -g main.go -o docs --parseDependency --parseInternal --propertyStrategy pascalcase

# Regenerate the swagger spec then the typed frontend API client (orval).
# The generated client (frontend/src/lib/api/generated) is committed, so the
# production build needs no codegen — this target is for refreshing it.
api-generate: swagger
	npm run api:generate --prefix frontend
