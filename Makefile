.PHONY: install dev docker helm-test

install:
	go install ./cmd/cb

dev:
	./scripts/dev.sh

docker:
	./scripts/docker-build-and-push.sh $(ARGS)

helm-test:
	./scripts/helm-test.sh
