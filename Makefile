test:
	TEST_DATABASE_URL=postgres://gostop:gostop@localhost:5432/gostop?sslmode=disable \
	go test -tags integration -count=1 -p 1 ./...

test-unit:
	go test ./internal/usecase/...

sqlc:
	sqlc generate

generate-phone-key:
	@openssl rand -base64 32
