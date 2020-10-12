.PHONY: test
test:
	go test -v -race ./...

.PHONY: test-bench
test-bench:
	go test -bench=.

.PHONY: test-cover
test-cover:
	go test -coverprofile=coverage.out

.PHONY: view-cover
view-cover:
	go tool cover -html=coverage.out
