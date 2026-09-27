.PHONY: test integration-test

test:
	go test ./...

# Starts the real server per scenario and prints the single- vs multi-acquirer
# acceptance report. Set OUT=<dir> to also save logs and analytics as evidence.
integration-test:
	go test -tags integration -count=1 -v ./integration-tests/ $(if $(OUT),-args -out=$(abspath $(OUT)))
