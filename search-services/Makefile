lint: protolint golint

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
	go generate ./...
	go test -race -covermode=atomic -coverprofile=cover.out ./api/... ./frontend/... ./search/... ./update/... ./words/...
	@grep -v -e 'mocks.go' -e 'main.go' cover.out > cover.filtered.out && mv cover.filtered.out cover.out 
	@go tool cover -func=cover.out | tail -n 1
	@mv cover.out ../

golint:
	golangci-lint run -E gocritic,gosec -v ./...
