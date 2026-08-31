dev:
	go run .
migrate:
	go run ./cmd/migrate up
migrate-status:
	go run ./cmd/migrate status
fmt:
	go fmt ./...
build-cdocs:
	cd docs/cdocs && FRONTEND_URL=https://botlist.site npm run build && cd ..
tests:
	CGO_ENABLED=0 go test -v -coverprofile=coverage.out ./...