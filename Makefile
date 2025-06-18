ADDR ?= 8080
ENV ?= dev
DB ?= postgres://pagesy:pagesy_password@localhost:5432/pagesy_db?sslmode=disable

build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/pagesy main.go

run: build
	./bin/pagesy

run-http: build
	./bin/pagesy http --addr=$(ADDR) --env=$(ENV)

migrate-create:
	migrate create -ext sql -dir internal/store/migrations -seq $(MIGRATION_NAME) 

migrate-up:
	migrate -path internal/store/migrations -database $(DB) up

migrate-down:
	migrate -path internal/store/migrations -database $(DB) down $(MIGRATION_NO)

migrate-force:
	migrate -path internal/store/migrations -database $(DB) force $(MIGRATION_VERSION)

docker-run-postgres:
	docker exec -it pagesy-postgres-1 psql -U pagesy -d pagesy_db

swagger:
	swag init

curl-healthz:
	curl -iX GET localhost:$(ADDR)/healthz

.PHONY: build run run-http migrate-create migrate-up migrate-down curl-healthz docker-run-postgres migrate-force
