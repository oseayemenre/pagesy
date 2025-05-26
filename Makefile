ADDR ?= 8080
ENV ?= dev

build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/pagesly main.go

run: build
	./bin/pagesly

run-http: build
	./bin/pagesly http --addr=$(ADDR) --env=$(ENV)

curl-healthz:
	curl -iX GET localhost:$(ADDR)/healthz


.PHONY: build run run-http curl-healthz
