# welcomebot Discord Bot

A clean, modern Discord bot built with Go following strict architectural principles.

## Architecture

- **Master Bot**: Handles Discord events, slash commands, and real-time interactions
- **Worker Bot**: Processes background tasks, async operations, and scheduled jobs
- **Distributed**: Master/Worker architecture with Redis task queue
- **Clean Code**: No god objects, strict typing, dependency injection

## Project Structure

```
├── cmd/              # Entry points
│   ├── master/       # Master bot
│   └── worker/       # Worker bot
├── internal/
│   ├── bot/          # Core bot logic
│   ├── features/     # Feature modules
│   ├── core/         # Shared services
│   └── shared/       # Shared types
├── deployments/      # K8s manifests
├── docs/             # Documentation
└── requirements/     # Feature requirements
```

## Development

### Prerequisites

- Go 1.24+
- PostgreSQL
- Redis
- golangci-lint

### Build

```bash
# Master bot
go build -o bin/master ./cmd/master

# Worker bot
go build -o bin/worker ./cmd/worker
```

### Linting

```bash
golangci-lint run
```

## Coding Guidelines

See [docs/CODING_GUIDELINES.md](docs/CODING_GUIDELINES.md) for comprehensive coding standards.

Key principles:
- No `interface{}` types
- Functions < 50 lines
- Files < 300 lines
- Explicit error handling
- Context-aware
- Dependency injection

## License

Private project.

