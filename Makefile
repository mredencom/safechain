.PHONY: test bench vet lint clean

test:
	go test -v -race -count=1 ./...

bench:
	go test -bench=. -benchmem ./...

vet:
	go vet ./...

cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

example:
	go run ./example/

clean:
	rm -f coverage.out coverage.html
