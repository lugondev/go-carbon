# Database Storage Example

This example demonstrates how to use the database storage layer in go-carbon with both MongoDB and PostgreSQL.

## Features Demonstrated

- Connecting to MongoDB or PostgreSQL
- Using DatasourceProcessor for single operations
- Using BatchDatasourceProcessor for bulk operations
- Querying stored data
- Automatic schema migrations (PostgreSQL)
- Automatic index creation (MongoDB)

## Prerequisites

### PostgreSQL

```bash
docker run -d \
  --name carbon-postgres \
  -e POSTGRES_USER=carbon \
  -e POSTGRES_PASSWORD=carbon123 \
  -e POSTGRES_DB=carbon_db \
  -p 5432:5432 \
  postgres:16-alpine
```

### MongoDB

```bash
docker run -d \
  --name carbon-mongo \
  -e MONGO_INITDB_ROOT_USERNAME=carbon \
  -e MONGO_INITDB_ROOT_PASSWORD=carbon123 \
  -p 27017:27017 \
  mongo:7
```

## Configuration

Edit `main.go` to switch between databases:

```go
cfg := &config.Config{
    Database: config.DatabaseConfig{
        Enabled: true,
        Type:    "postgres",  // or "mongodb"
        // ... connection details
    },
}
```

## Running

```bash
go run main.go
```

## What It Does

1. **Connects to database** using ConnectionManager
2. **Saves a single account** using DatasourceProcessor
3. **Batch saves multiple accounts** using BatchDatasourceProcessor
4. **Queries accounts** by owner
5. **Prints results** to console

## Output

```
{"level":"INFO","msg":"connected to database","type":"postgres"}
{"level":"DEBUG","msg":"account saved to database","pubkey":"TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA","slot":12345678}
{"level":"INFO","msg":"account saved successfully","pubkey":"TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA"}
{"level":"INFO","msg":"batch saved to database","count":5}
{"level":"INFO","msg":"batch saved successfully"}
{"level":"INFO","msg":"query results","count":6}
Account: TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA, Lamports: 1000000, Slot: 12345678
Account: 8z3t...x7q9, Lamports: 1000000, Slot: 12345678
Account: 9k2m...p4r6, Lamports: 2000000, Slot: 12345679
...
{"level":"INFO","msg":"example completed successfully"}
```

## Architecture

```
main.go
   │
   ├─► ConnectionManager (factory pattern)
   │       │
   │       ├─► MongoDB Repository
   │       └─► PostgreSQL Repository
   │
   ├─► DatasourceProcessor (single operations)
   └─► BatchDatasourceProcessor (bulk operations)
```

## Using in Your Pipeline

```go
import (
    "github.com/lugondev/go-carbon/internal/storage"
    "github.com/lugondev/go-carbon/internal/processor/database"
    _ "github.com/lugondev/go-carbon/internal/storage/mongo"
    _ "github.com/lugondev/go-carbon/internal/storage/postgres"
)

connMgr, _ := storage.NewConnectionManager(&cfg.Database)
repo, _ := connMgr.Connect(ctx)
defer connMgr.Close()

dbProcessor := database.NewDatasourceProcessor(repo, logger)
```

## See Also

- [Storage Layer Documentation](../../docs/storage.md)
- [PostgreSQL Migrations](../../internal/storage/postgres/migrations.go)
- [MongoDB Implementation](../../internal/storage/mongo/)
