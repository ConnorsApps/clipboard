.PHONY: install dev docker helm-test schema schema-check

install:
	go install ./cmd/cb

dev:
	./scripts/dev.sh

docker:
	./scripts/docker-build-and-push.sh $(ARGS)

helm-test:
	./scripts/helm-test.sh

schema:
	cd cmd/schema-gen && go run .

schema-check:
	cd cmd/schema-gen && go vet ./... && go test ./... && go run . -check
