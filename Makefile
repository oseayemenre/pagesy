ADDR ?= 8080
ENV ?= dev
DB ?= postgres://pagesy:pagesy_password@localhost:5432/pagesy_db?sslmode=disable

build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/pagesy cmd/*.go

run: build
	./bin/pagesy

run-http: build
	./bin/pagesy http --addr=$(ADDR) --env=$(ENV)

migrate-create:
	migrate create ext -sql -dir internal/store/migrations -seq $(MIGRATION_NAME) 

migrate-up:
	migrate -path internal/store/migrations -database $(DB) up

migrate-down:
	migrate -path internal/store/migrations -database $(DB) down $(MIGRATION_NO)

migrate-force:
	migrate -path internal/store/migrations -database $(DB) force $(MIGRATION_VERSION)

docker-run-postgres:
	docker exec -it pagesy-postgres-1 psql -U pagesy -d pagesy_db

curl-healthz:
	curl -iX GET localhost:$(ADDR)/healthz

curl-upload-file:
	curl -X POST localhost:8080/api/v1/books \
		-F "name=Book 1" \
		-F "description=Book 1 description" \
		-F "genre=Romance" \
		-F "genre=Mystery" \
		-F "language=Chinese" \
		-F "language=English" \
		-F "release_schedule_day=Sunday" \
		-F "release_schedule_chapter=2" \
		-F "release_schedule_day=Tuesday" \
		-F "release_schedule_chapter=3" \
		-F "chapter_draft=draft chapter 1" \
		-F "book_cover=@$(FILEPATH)"

.PHONY: build run run-http migrate-create migrate-up migrate-down curl-healthz curl-upload-file docker-run-postgres migrate-force
