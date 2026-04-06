APP_NAME := nrcc

.PHONY: build frontend-build dev run

build:
	go build -o bin/$(APP_NAME) .

frontend-build:
	cd frontend && npm install && npm run build

run:
	go run .

dev:
	@echo "Run 'make frontend-build' in one terminal and 'make run' in another once dependencies are installed."
