# MySQL Storage Example

This example demonstrates how to use the MySQL database storage layer in go-carbon.

## Features Demonstrated

- ✅ Connecting to MySQL database
- ✅ Using DatasourceProcessor for single operations  
- ✅ Using BatchDatasourceProcessor for bulk operations
- ✅ Querying stored data
- ✅ Automatic schema migrations
- ✅ Connection pooling configuration

## Prerequisites

### MySQL Database

You can run MySQL using Docker:

```bash
docker run -d \
  --name carbon-mysql \
  -e MYSQL_ROOT_PASSWORD=rootpass \
  -e MYSQL_DATABASE=carbon_db \
  -e MYSQL_USER=carbon \
  -e MYSQL_PASSWORD=carbon123 \
  -p 3306:3306 \
  mysql:8.0
```

Wait a few seconds for MySQL to initialize, then verify the connection:

```bash
mysql -h 127.0.0.1 -P 3306 -u carbon -pcarbon123 carbon_db -e "SELECT 1"
```

## Configuration

The example uses programmatic configuration:

```go
cfg := &config.Config{
    Database: config.DatabaseConfig{
        Enabled: true,
        Type:    "mysql",
        MySQL: config.MySQLConfig{
            Host:            "localhost",
            Port:            3306,
            User:            "carbon",
            Password:        "carbon123",
            Database:        "carbon_db",
            SSLMode:         "false",
            MaxOpenConns:    25,
            MaxIdleConns:    5,
            ConnMaxLifetime: 300,
        },
    },
}
```

You can also use a YAML config file (see `config.yaml`).

## Running

```bash
go run main.go
```

## What It Does

### 1. Single Operations
- Saves a single account (SPL Token Program)
- Demonstrates upsert behavior (ON DUPLICATE KEY UPDATE)

### 2. Batch Operations
- Creates 5 accounts with different lamports and slots
- Batches them together for efficient bulk insert
- Uses prepared statements for performance

### 3. Query Operations
- Queries all accounts by owner (System Program)
- Finds specific account by pubkey
- Demonstrates pagination with limit/offset

## Expected Output

```json
{"level":"INFO","msg":"connected to MySQL database successfully"}
{"level":"INFO","msg":"=== Demonstrating Single Operations ==="}
{"level":"INFO","msg":"account saved successfully","pubkey":"TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA","lamports":1000000,"slot":12345678}
{"level":"INFO","msg":"=== Demonstrating Batch Operations ==="}
{"level":"INFO","msg":"batch operations completed","count":5}
{"level":"INFO","msg":"=== Demonstrating Queries ==="}
{"level":"INFO","msg":"query results","count":6,"owner":"11111111111111111111111111111111"}
Account 1: TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA, Lamports: 1000000, Slot: 12345678
Account 2: 8z3t...x7q9, Lamports: 1000000, Slot: 12345678
Account 3: 9k2m...p4r6, Lamports: 2000000, Slot: 12345679
Account 4: 7h5n...s8w2, Lamports: 3000000, Slot: 12345680
Account 5: 6g4b...t9x3, Lamports: 4000000, Slot: 12345681
Account 6: 5f3a...u0y4, Lamports: 5000000, Slot: 12345682
{"level":"INFO","msg":"found specific account","pubkey":"TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA","lamports":1000000,"slot":12345678}
{"level":"INFO","msg":"MySQL storage example completed successfully"}
```

## Architecture

```
main.go
   │
   ├─► ConnectionManager
   │       └─► MySQL Repository (factory pattern)
   │
   ├─► DatasourceProcessor (single operations)
   │       └─► Saves one record at a time
   │
   └─► BatchDatasourceProcessor (bulk operations)
           ├─► Accumulates records in memory
           └─► Flushes with prepared statements
```

## Schema

The migrations automatically create these tables:

- `accounts` - Solana account data
- `transactions` - Transaction metadata and logs
- `instructions` - Instruction details
- `events` - Decoded program events
- `token_accounts` - SPL token account data
- `schema_migrations` - Migration version tracking

All tables use:
- InnoDB engine for ACID compliance
- UTF8MB4 character set for full Unicode support
- JSON columns for dynamic data
- Optimized indexes for common queries

## Performance Features

### Connection Pooling
```go
MaxOpenConns:    25   // Maximum open connections
MaxIdleConns:    5    // Idle connections to keep
ConnMaxLifetime: 300  // Recycle connections after 5 minutes
```

### Batch Operations
- Uses transactions for atomicity
- Prepared statements for efficiency
- Automatic batching when threshold reached

### Indexing
```sql
INDEX idx_accounts_owner (owner)
INDEX idx_accounts_slot (slot DESC)
INDEX idx_transactions_slot (slot DESC)
INDEX idx_events_program_id (program_id)
```

## Using in Your Pipeline

```go
import (
    "github.com/lugondev/go-carbon/internal/storage"
    "github.com/lugondev/go-carbon/internal/processor/database"
    _ "github.com/lugondev/go-carbon/internal/storage/mysql"
)

// Connect to MySQL
connMgr, _ := storage.NewConnectionManager(&cfg.Database)
repo, _ := connMgr.Connect(ctx)
defer connMgr.Close()

// Single operations
dbProcessor := database.NewDatasourceProcessor(repo, logger)
dbProcessor.ProcessAccountUpdate(ctx, accountUpdate)

// Batch operations (recommended for high throughput)
batchProcessor := database.NewBatchDatasourceProcessor(repo, logger, 100)
batchProcessor.ProcessAccountUpdate(ctx, accountUpdate)
batchProcessor.FlushAll(ctx)
```

## Troubleshooting

### Connection refused
- Make sure MySQL is running: `docker ps | grep carbon-mysql`
- Check port is exposed: `docker port carbon-mysql`

### Access denied for user
- Verify credentials in MySQL:
  ```bash
  docker exec -it carbon-mysql mysql -u root -prootpass \
    -e "SELECT user, host FROM mysql.user WHERE user='carbon'"
  ```

### Table doesn't exist
- Migrations run automatically on first connect
- Check migration status:
  ```bash
  docker exec -it carbon-mysql mysql -u carbon -pcarbon123 carbon_db \
    -e "SELECT * FROM schema_migrations"
  ```

### Slow queries
- Enable slow query log in MySQL
- Add indexes for your specific query patterns
- Increase connection pool size

## See Also

- [Database Documentation](../../docs/database.md)
- [MySQL Implementation](../../internal/storage/mysql/)
- [Storage Architecture](../../docs/architecture.md#storage-layer)
- [PostgreSQL Example](../database-storage/) (compare implementations)
