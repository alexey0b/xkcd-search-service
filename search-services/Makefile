lint: protolint golint

security: gosec

protobuf:
	protoc --go_out=. --go_opt=paths=source_relative \
               --go-grpc_out=. --go-grpc_opt=paths=source_relative \
               proto/words/words.proto
	protoc --go_out=. --go_opt=paths=source_relative \
               --go-grpc_out=. --go-grpc_opt=paths=source_relative \
               proto/update/update.proto
	protoc --go_out=. --go_opt=paths=source_relative \
               --go-grpc_out=. --go-grpc_opt=paths=source_relative \
               proto/search/search.proto

protolint:
	protolint .

unit-tests:
	go test ./api/... ./frontend/... ./search/... ./update/... ./words/...

cover:
	go test -coverprofile=cover.out ./api/... ./frontend/... ./search/... ./update/... ./words/...
	@grep -v -e 'mocks.go' -e 'main.go' cover.out > cover.filtered.out
	@go tool cover -html=cover.filtered.out -o cover.html
	@go tool cover -func=cover.filtered.out | tail -n 1
	@cat cover.filtered.out > cover.out
	@rm cover.filtered.out

gosec:
	gosec -exclude-generated ./...

golint:
	golangci-lint run -E gocritic -v ./...
