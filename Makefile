.PHONY: up seed down test
up:   ; docker compose up -d --build
seed: ; docker compose --profile seed run --rm seed
down: ; docker compose down
test: ; cd worker && npm test && cd ../backend && go test ./... && cd ../frontend && npm test
