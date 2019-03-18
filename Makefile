
build: fmt compile

fmt:
	gofmt -s -w message/*.go
	gofmt -s -w nats-mq-bridge/*.go
	gofmt -s -w server/conf/*.go
	gofmt -s -w server/core/*.go
	gofmt -s -w server/logging/*.go

	gofmt -s -w perf/encodingperf/*.go
	gofmt -s -w perf/fullmq2nats/*.go
	gofmt -s -w perf/multiplemq2nats/*.go
	gofmt -s -w perf/preloadmq2nats/*.go

	goimports -w message/*.go
	goimports -w nats-mq-bridge/*.go
	goimports -w server/conf/*.go
	goimports -w server/core/*.go
	goimports -w server/logging/*.go

	goimports -w perf/encodingperf/*.go
	goimports -w perf/fullmq2nats/*.go
	goimports -w perf/multiplemq2nats/*.go
	goimports -w perf/preloadmq2nats/*.go

compile:
	go build ./...

cover: test
	go tool cover -html=./coverage.out

test:
	go vet ./...
	rm -rf ./cover.out
	go test -coverpkg=./... -coverprofile=./cover.out ./...

