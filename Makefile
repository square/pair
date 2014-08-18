all: pair

clean:
	rm -f pair

pair: pair.go
	@go fmt pair.go
	go build pair.go

.PHONY: all clean
