.PHONY: build test clean

build:
	docker-compose up -d --build

test:
	go test -v ./...

clean:
	docker-compose down
