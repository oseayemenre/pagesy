ADDR ?= 8080
ENV ?= dev

build:
	go build -o bin/pagesly cmd/*.go

run: build
	./bin/pagesly

run-http: build
	./bin/pagesly http --addr=$(ADDR) --env="$(ENV)"

curl-healthz:
	curl -iX GET localhost:$(ADDR)/healthz


.PHONY: build run run-http curl-healthz
