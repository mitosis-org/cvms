# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

CVMS (Cosmos Validator Monitoring Service) is a Go-based monitoring system for validators in the Cosmos ecosystem. It provides integrated monitoring for validators and network maintainers across multiple Cosmos app chains, consolidating metrics like slashing uptime, oracle status, bridge status, and other chain-specific duties.

## Key Architecture Components

### Two Main Components
1. **Exporter**: Provides current on-chain status with integrated metrics regardless of app chain specifics
2. **Indexer**: Provides historical status tracking (e.g., vote status, proposed blocks)

### Operating Modes
- **Validator Mode**: Monitors specific whitelisted validators (configure with `moniker=['Cosmostation1', 'Cosmostation2']`)
- **Network Mode**: Monitors all validators in the network (configure with `moniker=['all']`)

### Core Package Structure
- `cmd/cvms`: Main entry point and CLI commands
- `internal/app/exporter`: Exporter application logic
- `internal/app/indexer`: Indexer application logic  
- `internal/packages/`: Chain-specific monitoring packages organized by duty type:
  - `health/`: Block monitoring
  - `consensus/`: Uptime tracking
  - `duty/`: Chain-specific duties (oracle, eventnonce, axelar-evm, yoda)
  - `utility/`: Balance and upgrade monitoring
  - `babylon/`: Babylon-specific packages
  - `axelar/`: Axelar-specific packages
  - `celestia/`: Celestia-specific packages
- `internal/common/`: Shared utilities, API clients, parsers, and types
- `internal/helper/`: Helper utilities (config, grpc, logging, SDK utilities)

### Database
- Uses PostgreSQL for indexer data storage
- Database models and repositories in `internal/common/indexer/`

## Development Commands

### Build and Install
```bash
make build           # Build binary to ./bin/cvms
make install         # Install binary to $GOPATH/bin/cvms
make clean          # Clean build artifacts
```

### Running CVMS
```bash
# Docker (recommended)
docker compose up --build -d

# Local development - Exporter
make start-exporter CONFIG_PATH=./config.yaml LOG_LEVEL=info LOG_COLOR_DISABLE=false

# Local development - Indexer  
make start-indexer CONFIG_PATH=./config.yaml LOG_LEVEL=info LOG_COLOR_DISABLE=false

# Run specific package only
make start-exporter-specific-package PACKAGE=oracle
make start-indexer-specific-package PACKAGE=voteindexer
```

### Testing
```bash
# Test all packages
make test-pkg-all

# Test specific packages
make test-pkg-block
make test-pkg-uptime  
make test-pkg-oracle
make test-pkg-eventnonce
make test-pkg-balance
make test-pkg-upgrade
make test-pkg-yoda
make test-pkg-axelarevm
make test-pkg-babylon-fp
```

### Linting
```bash
make ci              # Run all golangci-lint linters
make lint <linter>   # Run specific linter
```

### Database Operations
```bash
make reset-db        # Reset indexer database
make migration       # Run database migrations
```

### Configuration Management
```bash
# Copy and modify example config
cp .resource/example-validator-config.yaml config.yaml  # For validator mode
cp .resource/example-network-config.yaml config.yaml     # For network mode

# Sort support chains list
make sort_support_chains
```

## Configuration

### Chain Support
- Supported chains are defined in `docker/cvms/support_chains.yaml`
- Add custom chains in `docker/cvms/custom_chains.yaml`
- Each chain config includes endpoints (RPC, API, gRPC) and tracking addresses

### Environment Variables
- Configure via `.env` file (copy from `.resource/.env.example`)
- Controls Prometheus, Alertmanager settings, and CVMS log level

## Monitoring Stack

### Metrics
- All metrics prefixed with `cvms_<package>_<metric>`
- Exposed on port 9090 (exporter) and 9300 (indexer) by default
- Prometheus scrapes metrics, Grafana visualizes them

### Dashboards and Alerts
- Sample Grafana dashboards in `docker/grafana/provisioning/dashboards/`
- Alert rules in `docker/prometheus/rules/`

## Important Conventions

### Chain ID Format
- Chain IDs with hyphens are converted to underscores in table names (e.g., `cosmoshub-4` → `cosmoshub_4`)
- Use correct chain_id in config as it's the key to find applicable packages

### Error Handling
- Uses custom error types in `internal/common/errors.go`
- Implements exponential backoff for retries (`internal/helper/backoff.go`)

### Logging
- Structured logging with logrus
- Configure log level and color via environment variables

## Docker Deployment

The project includes a complete Docker Compose stack with:
- CVMS exporter and indexer services
- PostgreSQL database
- Prometheus for metrics collection
- Grafana for visualization  
- Alertmanager for alerting
- Loki and Promtail for log aggregation
- Flyway for database migrations