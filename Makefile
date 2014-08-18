all: pair test

clean:
	rm -f pair

pair: pair.go
	@go fmt pair.go
	go build

test: pair.go pair_test.go
	go test

.PHONY: all clean test
