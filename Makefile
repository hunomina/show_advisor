.PHONY: *

up:
	docker compose up -d --build

down:
	docker compose down

clean:
	docker compose down -v

logs:
	docker compose logs -f backend

pull-model:
	docker compose exec ollama ollama pull nomic-embed-text

ingest:
	docker compose run --rm backend ingest --csv /data/imdb_tvshows.csv

dashboard:
	open http://localhost:6333/dashboard

ui:
	open http://localhost:3000

dev-serve:
	go run ./cmd/showadvisor serve

dev-ingest:
	go run ./cmd/showadvisor ingest

dev-ui:
	cd frontend && npm run dev
