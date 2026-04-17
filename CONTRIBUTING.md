# Contributing to safechain

Thanks for your interest in contributing!

## How to contribute

1. Fork the repo
2. Create a feature branch (`git checkout -b feature/my-feature`)
3. Make your changes
4. Run tests: `go test -v -race ./...`
5. Run benchmarks: `go test -bench=. -benchmem ./...`
6. Commit and push
7. Open a Pull Request

## Guidelines

- All public functions must have godoc comments
- All public functions must have tests (including nil-path tests)
- Keep zero allocations on the happy path
- Run `go vet ./...` before submitting

## Reporting Issues

Open an issue on GitHub with:
- Go version (`go version`)
- OS and architecture
- Minimal reproduction code
