
build: fmt check compile

fmt:
	misspell -locale US .
	gofmt -s -w message/*.go
	gofmt -s -w nats-mq/*.go
	gofmt -s -w nats-mq/conf/*.go
	gofmt -s -w nats-mq/core/*.go
	gofmt -s -w nats-mq/logging/*.go
	gofmt -s -w performance/encodingperf/*.go
	gofmt -s -w performance/fullmq2nats/*.go
	gofmt -s -w performance/multiplemq2nats/*.go
	gofmt -s -w performance/preloadmq2nats/*.go
	goimports -w message/*.go
	goimports -w nats-mq/*.go
	goimports -w nats-mq/conf/*.go
	goimports -w nats-mq/core/*.go
	goimports -w nats-mq/logging/*.go
	goimports -w performance/encodingperf/*.go
	goimports -w performance/fullmq2nats/*.go
	goimports -w performance/multiplemq2nats/*.go
	goimports -w performance/preloadmq2nats/*.go

check:
	go vet ./...
	staticcheck ./...

update:
	go get -u honnef.co/go/tools/cmd/staticcheck
	go get -u github.com/client9/misspell/cmd/misspell
  
compile:
	go build ./...

cover: test
	go tool cover -html=./coverage.out

test: check
	rm -rf ./cover.out
	go test -coverpkg=./... -coverprofile=./cover.out ./...

