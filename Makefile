.PHONY: up down test logs smoke psql reset

BASE ?= http://localhost:9091

up:
	docker compose up -d --build
	curl -fsS --retry 20 --retry-delay 1 --retry-connrefused $(BASE)/health
	curl -fsS $(BASE)/readyz

down:
	docker compose down

test:
	go vet ./...
	go test -race ./...

logs:
	docker compose logs -f --tail=100 app

smoke:
	BASE=$(BASE) bash scripts/smoke.sh

psql:
	docker compose exec db sh -c 'psql -U "$$POSTGRES_USER" -d "$$POSTGRES_DB"'

reset:
	@echo "Удаляю контейнеры И данные БД"
	docker compose down -v