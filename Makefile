.PHONY: build run docker-build docker-run docker-clean

build:
	@echo "Building..."
	@go build -o bin/geocoding-api ./cmd/api

run:
	@echo "Running..."
	@go run ./cmd/api/main.go

docker-build:
	@echo "Building Docker container..."
	@docker build -t geocoding-api .

docker-run:
	@echo "Running Docker container..."
	@docker run -p 8080:8080 --name geocoding-api-container geocoding-api

docker-clean:
	@echo "Cleaning up Docker containers and images..."
	@docker stop geocoding-api-container 2>/dev/null || true
	@docker rm geocoding-api-container 2>/dev/null || true
	@docker rmi geocoding-api 2>/dev/null || true
