test:
	TEST_DATABASE_URL=postgres://gostop:gostop@localhost:5432/gostop?sslmode=disable \
	go test -tags integration -count=1 -p 1 ./...

test-unit:
	go test ./internal/usecase/...

sqlc:
	sqlc generate

build-web:
	npm ci --prefix frontend && npm run build --prefix frontend

dev:
	@echo "Go :8080 + Vite :5173 (proxying /api). Ctrl-C stops both."
	@( go run . & echo $$! > /tmp/gostop-go.pid ; npm run dev --prefix frontend ; kill `cat /tmp/gostop-go.pid` )

test-e2e: build-web
	npm run test:e2e

swagger-install:
	go install github.com/swaggo/swag/cmd/swag@latest

swagger:
	swag init -g main.go -o docs --parseDependency --parseInternal --propertyStrategy pascalcase
