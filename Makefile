.PHONY: build test run clean

build:
	go build -o authcli ./cmd/authcli

test:
	go test -v ./...

run: build
	./authcli

docker-build:
	docker compose build

docker-run:
	docker compose run --rm auth-cli

clean:
	rm -f authcli
	rm -rf /data/auth.db
