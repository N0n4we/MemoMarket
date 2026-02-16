.PHONY: build run clean dev

build:
	CGO_ENABLED=1 go build -o memomarket-server .

run: build
	./memomarket-server

dev:
	go run .

clean:
	rm -f memomarket-server
	rm -rf data/

docker-build:
	docker build -t memomarket-backend .

docker-run:
	docker run -p 8080:8080 -v memomarket-data:/app/data memomarket-backend
