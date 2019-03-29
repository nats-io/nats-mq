
build: fmt check compile

fmt:
	misspell -locale US .
	gofmt -s -w message/*.go
	gofmt -s -w nats-mq/*.go
	gofmt -s -w nats-mq/conf/*.go
	gofmt -s -w nats-mq/core/*.go
	gofmt -s -w nats-mq/logging/*.go
	gofmt -s -w performance/full/*.go
	gofmt -s -w performance/queues/*.go
	gofmt -s -w performance/full_testenv/*.go
	gofmt -s -w performance/multiqueue_testenv/*.go
	gofmt -s -w performance/singlequeue_testenv/*.go
	goimports -w message/*.go
	goimports -w nats-mq/*.go
	goimports -w nats-mq/conf/*.go
	goimports -w nats-mq/core/*.go
	goimports -w nats-mq/logging/*.go
	goimports -w performance/encodingperf/*.go
	goimports -w performance/full/*.go
	goimports -w performance/queues/*.go
	goimports -w performance/full_testenv/*.go
	goimports -w performance/multiqueue_testenv/*.go
	goimports -w performance/singlequeue_testenv/*.go

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

