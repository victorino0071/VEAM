# Contributing to VEAM

First off, thank you for considering contributing to **VEAM**! It's people like you that make VEAM such a great tool for the community.

## How to Contribute

### Reporting Bugs
- Open an issue on GitHub.
- Describe the bug and how to reproduce it.
- Include environment details (Go version, OS, etc.).

### Suggesting Enhancements
- Open an issue on GitHub.
- Clearly describe the feature and why it would be useful.

### Pull Requests
1. Fork the repository.
2. Create a new branch (`git checkout -b feature/my-new-feature`).
3. Make your changes.
4. Run tests (`go test ./...`) to ensure everything is working.
5. Commit your changes (`git commit -am 'Add some feature'`).
6. Push to the branch (`git push origin feature/my-new-feature`).
7. Create a new Pull Request.

## Code of Conduct
Please be respectful and professional in all interactions.

## Development Setup
1. Clone your fork.
2. Run `go mod tidy` to install dependencies.
3. Use `docker-compose` (if available) or a local Postgres instance for integration tests.

Happy coding!
