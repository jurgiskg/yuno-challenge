.PHONY: test run integration-test integration-test-deployed

# Load local settings from .env (see .env.example) and export them to the
# commands below. Values in .env override the shell environment; override a
# single run with `make <target> KEY=value`.
ifneq (,$(wildcard .env))
include .env
export $(shell sed -n 's/^\([A-Za-z_][A-Za-z0-9_]*\)=.*/\1/p' .env)
endif

# ./... skips directories named testdata, so list it explicitly.
test:
	go test ./... ./testdata/

# Runs the service with the settings from .env.
run:
	go run .

# Starts the real server per scenario and writes the single- vs multi-acquirer
# acceptance report to integration-tests/report.html. Set OUT=<dir> to also save
# logs and analytics as evidence. Always runs locally, even when .env sets
# INTEGRATION_BASE_URL.
integration-test:
	INTEGRATION_BASE_URL= go test -tags integration -count=1 -v ./integration-tests/ $(if $(OUT),-args -out=$(abspath $(OUT)))

# Runs the integration test against INTEGRATION_BASE_URL with INTEGRATION_API_KEY.
integration-test-deployed:
	@test -n "$(INTEGRATION_BASE_URL)" || { echo "INTEGRATION_BASE_URL is not set (see .env.example)"; exit 1; }
	@test -n "$(INTEGRATION_API_KEY)" || { echo "INTEGRATION_API_KEY is not set (see .env.example)"; exit 1; }
	go test -tags integration -count=1 -v ./integration-tests/ $(if $(OUT),-args -out=$(abspath $(OUT)))
