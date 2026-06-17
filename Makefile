PORT ?= 8765

.PHONY: integration-test
integration-test:
	PORT=$(PORT) ./scripts/run-integration-tests.sh $(ARGS)
