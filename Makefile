up:
	docker compose up -d --build
	curl -i "localhost:9091/readyz"     
	curl -i "localhost:9091/health"


down:
	docker compose down


test:
	go test ./...
	go test -race ./...


logs:
	docker ps
	docker compose logs app