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

golint:
	golangci-lint run -E gocritic -v ./...

test:
	go test -coverprofile=cover.out -covermode=atomic $$( go list ./... | grep -v proto )
	go tool cover -html=cover.out -o cover.html
