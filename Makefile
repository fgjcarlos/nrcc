APP_NAME := nrcc

.PHONY: build frontend-build release-package dev run

build:
	go build -o bin/$(APP_NAME) .

frontend-build:
	cd frontend && npm ci && npm run build

release-package:
	./scripts/package-release.sh

run:
	go run .

dev:
	@echo "Run 'make frontend-build' in one terminal and 'make run' in another once dependencies are installed."
