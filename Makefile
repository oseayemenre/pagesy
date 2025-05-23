build:
	go build -o bin/pagesly cmd/*.go

run: build
	./bin/pagesly

.PHONY: build run
