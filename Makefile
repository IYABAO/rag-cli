.PHONY: build run up down clean

build:
	go build -o rag-cli.exe .

up:
	docker compose up -d

down:
	docker compose down

ingest:
	./rag-cli.exe --local --qdrant ingest ./notes.md

ask:
	./rag-cli.exe --local --qdrant ask "$(q)"

clean:
	rm -f rag-cli.exe rag_index.json
