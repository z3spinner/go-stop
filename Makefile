test:
	TEST_DATABASE_URL=postgres://gostop:gostop@localhost:5432/gostop?sslmode=disable \
	go test -tags integration -count=1 -p 1 ./...

test-unit:
	go test ./internal/usecase/...

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

test-e2e: build-web
	npm run test:e2e

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
