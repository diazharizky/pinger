.PHONY: run up down build clean

run:
	source .env &&\
	go run cmd/pinger/*.go

help:
	source .env &&\
	go run cmd/pinger/*.go help

up:
	cd build/docker &&\
	docker-compose up -d pushgateway

down:
	cd build/docker &&\
	docker-compose down

build:
	for dir in `find ./cmd -name main.go -type f`; do \
		go build -v -o "bin/$$(basename $$(dirname $$dir))" "$$(dirname $$dir)"; \
	done

clean:
	rm -rf bin;
