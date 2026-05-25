.PHONY: install dev docker helm

install:
	go install ./cmd/cb

dev:
	./scripts/dev.sh

docker:
	./scripts/docker-build-and-push.sh $(ARGS)

helm:
	./scripts/package-chart.sh
