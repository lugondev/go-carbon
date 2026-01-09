# Database Storage Layer

Go-Carbon provides a flexible database storage layer that supports multiple databases through a unified repository pattern.

## Supported Databases

- **MongoDB** - Document-based NoSQL database
- **PostgreSQL** - Relational SQL database with JSONB support
- **MySQL** - Popular relational SQL database with JSON support

## Architecture

```
┌─────────────────────────────────────────────────────┐
│              ConnectionManager                      │
│         (Factory + Connection Pooling)              │
└───────────────┬─────────────────────────────────────┘
                │
                ├─► MongoDB Repository
                │   ├─ Auto-creates indexes
                │   ├─ Batch operations with BulkWrite
                │   └─ Connection pooling
                │
                ├─► PostgreSQL Repository
                │   ├─ Schema migrations
                │   ├─ Batch operations with pgx.Batch
                │   └─ Connection pooling with pgxpool
                │
                └─► MySQL Repository
                    ├─ Schema migrations
                    ├─ Batch operations with transactions
                    └─ Connection pooling with database/sql
```

## Quick Start

### 1. Import Database Packages

```go
import (
    "github.com/lugondev/go-carbon/internal/storage"
    "github.com/lugondev/go-carbon/internal/processor/database"
    
    _ "github.com/lugondev/go-carbon/internal/storage/mongo"
    _ "github.com/lugondev/go-carbon/internal/storage/postgres"
    _ "github.com/lugondev/go-carbon/internal/storage/mysql"
)
```

### 2. Configure Database

```go
cfg := &config.Config{
    Database: config.DatabaseConfig{
        Enabled: true,
        Type:    "postgres", // or "mongodb" or "mysql"
        
        Postgres: config.PostgresConfig{
            Host:            "localhost",
            Port:            5432,
            User:            "carbon",
            Password:        "carbon123",
            Database:        "carbon_db",
            SSLMode:         "disable",
            MaxOpenConns:    25,
            MaxIdleConns:    5,
            ConnMaxLifetime: 300,
        },
        
        MongoDB: config.MongoDBConfig{
            URI:            "mongodb://localhost:27017",
            Database:       "carbon_db",
            MaxPoolSize:    100,
            MinPoolSize:    10,
            ConnectTimeout: 10,
        },
        
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

### 3. Connect to Database

```go
connMgr, err := storage.NewConnectionManager(&cfg.Database)
if err != nil {
    log.Fatal(err)
}

repo, err := connMgr.Connect(ctx)
if err != nil {
    log.Fatal(err)
}
defer connMgr.Close()
```

### 4. Use in Your Application

#### Single Operations

```go
processor := database.NewDatasourceProcessor(repo, logger)

accountUpdate := &datasource.AccountUpdate{
    Pubkey:  pubkey,
    Account: account,
    Slot:    slot,
}

if err := processor.ProcessAccountUpdate(ctx, accountUpdate); err != nil {
    log.Fatal(err)
}
```

#### Batch Operations

```go
batchProcessor := database.NewBatchDatasourceProcessor(repo, logger, 100)

for _, update := range updates {
    if err := batchProcessor.ProcessAccountUpdate(ctx, update); err != nil {
        log.Fatal(err)
    }
}

if err := batchProcessor.FlushAll(ctx); err != nil {
    log.Fatal(err)
}
```

## Repository Interface

All databases implement the unified `Repository` interface:

```go
type Repository interface {
    Accounts() AccountRepository
    Transactions() TransactionRepository
    Instructions() InstructionRepository
    Events() EventRepository
    TokenAccounts() TokenAccountRepository
    Close() error
}
```

## Domain Models

### AccountModel
Stores Solana account data with owner, lamports, data, etc.

### TransactionModel
Stores transaction details including signature, slot, success status, logs, compute units.

### InstructionModel
Stores instruction data with program ID, accounts, and instruction index.

### EventModel
Stores decoded events from programs with JSONB data.

### TokenAccountModel
Stores SPL token account information including mint, owner, amount, decimals.

## MongoDB Features

### Automatic Index Creation

MongoDB automatically creates indexes on:
- `accounts`: pubkey, owner, slot
- `transactions`: signature, slot, success, block_time
- `instructions`: signature, program_id
- `events`: signature, program_id, event_name, slot
- `token_accounts`: address, mint, owner, slot

### Upsert Operations

Account and token account saves use upsert to handle updates:

```go
repo.Accounts().Save(ctx, accountModel)  // Creates or updates
```

### Batch Operations

Uses MongoDB's `BulkWrite` for efficient batch operations:

```go
repo.Accounts().SaveBatch(ctx, accounts)
```

## PostgreSQL Features

### Schema Migrations

PostgreSQL uses a migration system for schema evolution:

```go
migrator := postgres.NewMigrator(pool)

migrator.Up(ctx)         // Apply pending migrations
migrator.Down(ctx, 1)    // Rollback 1 migration
migrator.Status(ctx)     // Check migration status
```

Migrations are automatically applied when connecting.

### Schema Versioning

Tracks applied migrations in `schema_migrations` table:

```sql
CREATE TABLE schema_migrations (
    version INT PRIMARY KEY,
    description TEXT NOT NULL,
    applied_at TIMESTAMP NOT NULL DEFAULT NOW()
);
```

### Batch Operations

Uses `pgx.Batch` for efficient bulk operations:

```go
repo.Transactions().SaveBatch(ctx, transactions)
```

## MySQL Features

### Schema Migrations

MySQL uses the same migration system as PostgreSQL:

```go
migrator := mysql.NewMigrator(db)

migrator.Up(ctx)         // Apply pending migrations
migrator.Down(ctx, 1)    // Rollback 1 migration
migrator.Status(ctx)     // Check migration status
```

Migrations are automatically applied when connecting to ensure schema is up-to-date.

### Database Schema

MySQL tables are created with:
- **InnoDB engine** for ACID compliance and foreign key support
- **UTF8MB4 character set** for full Unicode support (including emojis)
- **JSON columns** for dynamic data (account_keys, log_messages, event_data)
- **Optimized indexes** for common query patterns

Example schema:

```sql
CREATE TABLE accounts (
    pubkey VARCHAR(44) PRIMARY KEY,
    owner VARCHAR(44) NOT NULL,
    lamports BIGINT UNSIGNED NOT NULL,
    slot BIGINT UNSIGNED NOT NULL,
    executable BOOLEAN NOT NULL,
    rent_epoch BIGINT UNSIGNED NOT NULL,
    data LONGBLOB,
    account_keys JSON,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_owner (owner),
    INDEX idx_slot (slot)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

### Batch Operations

Uses prepared statements with transactions for optimal performance:

```go
repo.Accounts().SaveBatch(ctx, accounts)
```

Internally uses:
- `database/sql` transaction API
- Prepared statement reuse within transaction
- `ON DUPLICATE KEY UPDATE` for efficient upserts
- Automatic error handling and rollback

### Connection Pooling

MySQL connection pool is highly configurable:

```go
mysql: config.MySQLConfig{
    MaxOpenConns:    25,  // Maximum concurrent connections
    MaxIdleConns:    5,   // Idle connections to keep alive
    ConnMaxLifetime: 300, // Recycle connections after 5 minutes
}
```

**Performance Tips:**
- Set `MaxOpenConns` to 2-3x your concurrent workload
- Keep `MaxIdleConns` lower to reduce idle connections
- Use `ConnMaxLifetime` to prevent stale connections (300-900s recommended)

### JSON Support

MySQL 5.7+ provides native JSON support:

```go
// Query using JSON functions
SELECT * FROM events 
WHERE JSON_CONTAINS(event_data, '"value"', '$.field')
```

**JSON Functions:**
- `JSON_EXTRACT(col, '$.path')` - Extract field
- `JSON_CONTAINS(col, 'value', '$.path')` - Check if contains
- `JSON_SEARCH(col, 'one', 'pattern')` - Search for pattern
- Generated columns for indexing JSON fields (if needed)

### Configuration Example

**YAML:**

```yaml
database:
  enabled: true
  type: mysql
  
  mysql:
    host: localhost
    port: 3306
    user: carbon
    password: carbon123
    database: carbon_db
    ssl_mode: "false"  # "true", "false", "skip-verify", "preferred"
    max_open_conns: 25
    max_idle_conns: 5
    conn_max_lifetime: 300
```

**Go Code:**

```go
cfg := &config.MySQLConfig{
    Host:            "localhost",
    Port:            3306,
    User:            "carbon",
    Password:        "carbon123",
    Database:        "carbon_db",
    SSLMode:         "false",
    MaxOpenConns:    25,
    MaxIdleConns:    5,
    ConnMaxLifetime: 300,
}

repo, err := mysql.NewMySQLRepository(ctx, cfg)
```

### Docker Setup

```bash
docker run -d \
  --name carbon-mysql \
  -e MYSQL_ROOT_PASSWORD=rootpass \
  -e MYSQL_DATABASE=carbon_db \
  -e MYSQL_USER=carbon \
  -e MYSQL_PASSWORD=carbon123 \
  -p 3306:3306 \
  mysql:8.0

# Wait for MySQL to be ready (30 seconds)
sleep 30

# Verify connection
docker exec -it carbon-mysql mysql -ucarbon -pcarbon123 carbon_db -e "SELECT 1"
```

### Performance Tips

1. **Use InnoDB Engine** (default in migrations)
   - ACID compliance
   - Foreign key support
   - Row-level locking

2. **Tune InnoDB Buffer Pool**
   ```sql
   SET GLOBAL innodb_buffer_pool_size = 2G;  -- 50-70% of available RAM
   ```

3. **Enable Query Cache** (for read-heavy workloads)
   ```sql
   SET GLOBAL query_cache_size = 64M;
   SET GLOBAL query_cache_type = ON;
   ```

4. **Use Batch Operations**
   ```go
   // Good: Batch insert 1000 accounts
   repo.Accounts().SaveBatch(ctx, accounts)
   
   // Bad: 1000 individual inserts
   for _, acc := range accounts {
       repo.Accounts().Save(ctx, acc)
   }
   ```

5. **Monitor Slow Queries**
   ```sql
   SET GLOBAL slow_query_log = 'ON';
   SET GLOBAL long_query_time = 2;
   ```

### Differences from PostgreSQL

| Feature              | PostgreSQL          | MySQL                      |
|---------------------|---------------------|----------------------------|
| JSON Type           | JSONB (binary)      | JSON (text-based)          |
| Upsert Syntax       | `ON CONFLICT`       | `ON DUPLICATE KEY UPDATE`  |
| Array Support       | Native arrays       | JSON arrays                |
| Driver              | pgx/v5              | go-sql-driver/mysql        |
| Connection Pool     | pgxpool             | database/sql               |
| Boolean Type        | `BOOLEAN`           | `BOOLEAN` (alias for `TINYINT(1)`) |
| Transaction Isolation | Higher isolation  | Lower isolation (default)  |
| Full-Text Search    | Built-in            | Built-in with FULLTEXT     |

### When to Use MySQL

**Choose MySQL if:**
- ✅ Team already familiar with MySQL administration
- ✅ Existing MySQL infrastructure and tooling
- ✅ Need proven reliability and wide community support
- ✅ Using MySQL-specific features (spatial data, full-text search)
- ✅ Simpler replication setup required

**Choose PostgreSQL if:**
- ✅ Need advanced JSON operations (JSONB with GIN indexes)
- ✅ Need native array support and array operations
- ✅ Want better concurrent write performance
- ✅ Need advanced indexing (GIN, GiST, SP-GiST)
- ✅ Require stricter ACID compliance

**Choose MongoDB if:**
- ✅ Fully document-oriented data model
- ✅ Need flexible schema evolution
- ✅ Horizontal scaling with sharding
- ✅ Geospatial queries with 2dsphere indexes

## Querying Data

### Find Accounts

```go
account, err := repo.Accounts().FindByPubkey(ctx, pubkey)

accounts, err := repo.Accounts().FindByOwner(ctx, owner, offset, limit)

recentAccounts, err := repo.Accounts().FindRecent(ctx, limit)
```

### Find Transactions

```go
tx, err := repo.Transactions().FindBySignature(ctx, signature)

txs, err := repo.Transactions().FindBySlot(ctx, slot, offset, limit)

successfulTxs, err := repo.Transactions().FindSuccessful(ctx, limit, offset)
```

### Find Instructions

```go
instructions, err := repo.Instructions().FindBySignature(ctx, signature)

programInstructions, err := repo.Instructions().FindByProgramID(ctx, programID, offset, limit)
```

### Find Events

```go
events, err := repo.Events().FindBySignature(ctx, signature)

programEvents, err := repo.Events().FindByProgramID(ctx, programID, offset, limit)

namedEvents, err := repo.Events().FindByEventName(ctx, eventName, offset, limit)
```

## Connection Pooling

Both databases use connection pooling for optimal performance:

### PostgreSQL
- Uses `pgxpool` with configurable pool size
- Health checks every minute
- Configurable connection lifetime

### MongoDB
- Uses MongoDB driver's built-in pooling
- Configurable min/max pool size
- Automatic connection management

## Error Handling

All repository methods return errors that can be checked:

```go
if err := repo.Accounts().Save(ctx, model); err != nil {
    if errors.Is(err, context.Canceled) {
        // Context was canceled
    } else {
        // Other error
        log.Printf("Failed to save: %v", err)
    }
}
```

## Performance Tips

### Use Batch Operations

For high-throughput scenarios, use batch processors:

```go
batchProcessor := database.NewBatchDatasourceProcessor(repo, logger, 1000)
```

### Adjust Connection Pool Size

Tune connection pool settings based on your workload:

```yaml
postgres:
  max_open_conns: 50    # Increase for high concurrency
  max_idle_conns: 10
  conn_max_lifetime: 300

mongodb:
  max_pool_size: 200    # Increase for high concurrency
  min_pool_size: 20
```

### Use Indexes Effectively

Ensure your queries use indexes:

```go
accounts, _ := repo.Accounts().FindByOwner(ctx, owner, 0, 100)
```

## Environment Variables

```bash
export CARBON_DB_TYPE=postgres
export CARBON_DB_HOST=localhost
export CARBON_DB_PORT=5432
export CARBON_DB_USER=carbon
export CARBON_DB_PASSWORD=carbon123
export CARBON_DB_NAME=carbon_db
```

## Docker Setup

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

### MySQL

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

## Adding New Database Support

To add support for a new database:

1. Create package in `internal/storage/yourdb/`
2. Implement `Repository` interface
3. Register factory in `init()`:

```go
func init() {
    storage.RegisterRepositoryFactory("yourdb", NewYourDBRepositoryFromConfig)
}
```

4. Import in your application:

```go
import _ "github.com/lugondev/go-carbon/internal/storage/yourdb"
```

## See Also

- [Example: Database Storage (PostgreSQL/MongoDB)](../examples/database-storage/)
- [Example: MySQL Storage](../examples/mysql-storage/)
- [MongoDB Implementation](../internal/storage/mongo/)
- [PostgreSQL Implementation](../internal/storage/postgres/)
- [MySQL Implementation](../internal/storage/mysql/)
- [Repository Pattern](../internal/storage/repository.go)
