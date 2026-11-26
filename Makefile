up:
	docker compose up -d --build
	curl -i "localhost:9091/health"
	curl -i "localhost:9091/readyz"     


down:
	docker compose down


test:
	go test ./...
	go test -race ./...


logs:
	docker ps
	docker compose logs app