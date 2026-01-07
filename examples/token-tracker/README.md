# Token Tracker Example

A comprehensive example demonstrating real-time SPL Token account monitoring using the go-carbon framework. Features include balance tracking, movement detection, mint filtering, and rich logging with detailed alerts.

## ✨ Features

- **🔍 Account Monitoring**: Track multiple token accounts simultaneously
- **📊 Balance Tracking**: Monitor token balance changes in real-time
- **🎯 Track by Mint**: Monitor ALL movements for specific token mints (new!)
- **🚨 Smart Alerts**: Configurable alerts for significant movements, new accounts, and state changes
- **📈 Rich Logging**: Human-readable amounts with decimals, emojis, and detailed movement info
- **🪙 Mint Supply Tracking**: Optional mint supply change monitoring
- **⚙️ Fully Configurable**: All settings managed via YAML config

## 📁 Project Structure

```
token-tracker/
├── main.go              # Pipeline setup and orchestration
├── config.go            # Configuration structures and parsing
├── decoder.go           # SPL Token account/mint decoders
├── tracker.go           # Token movement tracking with rich logging
├── mint_scanner.go      # Scan blockchain for accounts by mint
├── config.yaml          # Default configuration
└── config.track-by-mint.yaml  # Example: Track by mint mode
```

**Total**: ~40KB of clean, modular Go code

## 🚀 Quick Start

### 1. Configure

**Option A: Track Specific Accounts** (`config.yaml`)
```yaml
accounts:
  - "2wmVCSfPxGPjrnMMn7rchp4uaeoTqN39mXFC2zhPdri9"
  - "9RfZwn2Prux6QesG1Noo4HzMEBv3rPndJ2bN2Wwd6a7p"

track_by_mint:
  enabled: false
```

**Option B: Track All Movements for a Mint** (`config.track-by-mint.yaml`)
```yaml
accounts: []

track_by_mint:
  enabled: true
  mints:
    - "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v"  # USDC
  max_accounts: 100
```

### 2. Run

```bash
# Track specific accounts
go run . -config config.yaml

# Track all USDC movements
go run . -config config.track-by-mint.yaml

# Or build first
go build -o token-tracker .
./token-tracker -config config.yaml
```

## 📊 Output Examples

### Token Movement Detection

```
🚀 Starting go-carbon token tracker
👀 Monitoring accounts count=2
   [1] 2wmVCSfP...dri9
   [2] 9RfZwn2P...6a7p
✅ Pipeline started, monitoring for token movements...
💡 Press Ctrl+C to stop

🆕 NEW TOKEN ACCOUNT DETECTED
  account: 2wmVCSfP...
  mint: 8rUUP52B...
  owner: Cq7CPoJ3...
  balance: 1000.000000
  state: Initialized

⬇️ RECEIVED TOKEN MOVEMENT
  account: 2wmVCSfP...
  mint: 8rUUP52B...
  change: +500.000000
  prev_balance: 1000.000000
  balance: 1500.000000

⬆️ SENT TOKEN MOVEMENT
  account: 2wmVCSfP...
  mint: 8rUUP52B...
  change: -200.000000
  prev_balance: 1500.000000
  balance: 1300.000000

🚨 ALERT: Significant token movement received to
  📥 emoji
  amount: +10000.000000
  threshold: 1000000
```

### Emoji Legend

| Emoji | Meaning |
|-------|---------|
| 🚀 | Starting up |
| 👀 | Monitoring accounts |
| 🆕 | New account detected |
| ⬇️ | Token received |
| ⬆️ | Token sent |
| 🔄 | Account state changed |
| 🪙 | Mint discovered/updated |
| ⚡ | Tokens minted |
| 🔥 | Tokens burned |
| 🚨 | Alert triggered |
| 📥 | Incoming transfer (alert) |
| 📤 | Outgoing transfer (alert) |
| ❄️ | Account frozen |
| ⚠️ | Warning |
| ✅ | Success |
| 🛑 | Shutdown |

## ⚙️ Configuration Guide

### RPC Settings

```yaml
rpc:
  endpoint: "https://api.mainnet-beta.solana.com"
  poll_interval: 3  # seconds
  timeout: 30
```

**Recommended RPCs:**
- Development: `https://api.devnet.solana.com`
- Mainnet: Custom RPC (Helius, Quicknode, etc.)
- Track by Mint: Requires RPC with `getProgramAccounts` support

### Account Monitoring

```yaml
accounts:
  - "YourTokenAccount1..."
  - "YourTokenAccount2..."
```

Add token account addresses you want to monitor. Leave empty when using `track_by_mint` mode.

### Track by Mint (NEW!)

```yaml
track_by_mint:
  enabled: true
  mints:
    - "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v"  # USDC
    - "So11111111111111111111111111111111111111112"   # Wrapped SOL
  max_accounts: 100      # Safety limit
  refresh_interval: 60   # Seconds (future feature)
```

**When to use:**
- Track ALL movements for a specific token
- Monitor whale activity across entire token
- Analyze token distribution patterns
- No need to specify individual accounts

**Important:**
- Uses `getProgramAccounts` RPC call (may be rate-limited)
- Not all public RPCs support this
- Use paid RPC providers (Helius, Quicknode) for production

### Alert Configuration

```yaml
alerts:
  enabled: true
  threshold: 1000000        # Raw token amount
  alert_new_accounts: true
  alert_state_changes: true
```

**Threshold Examples:**
- USDC (6 decimals): `1000000` = 1 USDC
- SOL (9 decimals): `1000000000` = 1 SOL
- Adjust based on your token's decimals

### Mint Filtering (Optional)

```yaml
target_mints:
  enabled: true
  mints:
    - "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v"
```

**Use case:** When monitoring multiple accounts but only care about specific tokens.

### Logging

```yaml
logging:
  level: "info"      # debug, info, warn, error
  format: "text"     # text or json
```

- **debug**: Shows all RPC calls, decoder operations
- **info**: Movement detection, alerts (recommended)
- **warn**: Only alerts and warnings
- **error**: Only errors

### Experimental Features

```yaml
advanced:
  experimental:
    track_mint_supply: true   # Track supply changes
    decode_token_extensions: false  # Token-2022 support
    enable_caching: false
```

## 💡 Usage Patterns

### Pattern 1: Monitor Your Own Accounts

Track your wallet's token accounts:

```yaml
accounts:
  - "YourUSDCAccount..."
  - "YourSOLAccount..."

alerts:
  enabled: true
  threshold: 100000  # Alert on significant changes
```

### Pattern 2: Track Whale Activity

Monitor large holders of a specific token:

```yaml
track_by_mint:
  enabled: true
  mints:
    - "YourTokenMint..."
  max_accounts: 50  # Top 50 holders

alerts:
  threshold: 1000000000  # High threshold for whales
```

### Pattern 3: Multi-Token Portfolio

Track specific accounts across multiple tokens:

```yaml
accounts:
  - "Account1..."
  - "Account2..."
  - "Account3..."

target_mints:
  enabled: true
  mints:
    - "USDC_MINT"
    - "USDT_MINT"
    - "SOL_MINT"
```

### Pattern 4: Full Mint Analysis

Analyze all movements for a token:

```yaml
track_by_mint:
  enabled: true
  mints:
    - "YourTokenMint..."
  max_accounts: 100

advanced:
  experimental:
    track_mint_supply: true  # Also track minting/burning
```

## 🔧 Understanding Token Amounts

Token amounts are stored as raw values without decimal adjustment. The tracker automatically formats them based on mint decimals.

**Raw vs Formatted:**
```
Raw Amount: 1000000
Decimals: 6
Formatted: 1.000000 USDC

Raw Amount: 1000000000
Decimals: 9
Formatted: 1.000000 SOL
```

The tracker displays both:
- `balance`: Human-readable with decimals
- `balance_raw`: Raw u64 value

## 🐛 Troubleshooting

### "No accounts configured to monitor"

```
⚠️ No accounts configured to monitor
```

**Solution:** Add account addresses to `accounts` section or enable `track_by_mint`.

### "Failed to scan token accounts"

```
ERROR Failed to scan token accounts error="getProgramAccounts not supported"
```

**Solutions:**
- Use RPC provider that supports `getProgramAccounts`
- Try Helius, Quicknode, or Alchemy
- Switch to specific account monitoring mode

### "Account not owned by token program"

Account is not a token account (might be a regular wallet or program).

**Solution:** Ensure addresses are token accounts, not wallets.

### RPC Rate Limiting

```
WARN failed to get account info error="429 Too Many Requests"
```

**Solutions:**
- Increase `poll_interval` (default: 3s)
- Use paid RPC with higher limits
- Reduce number of monitored accounts

### High Memory Usage

When tracking many accounts with `track_by_mint`:

**Solutions:**
- Lower `max_accounts` limit
- Disable `track_mint_supply` if not needed
- Enable `enable_caching: true` (experimental)

## 🚀 Performance Tips

1. **Optimize Poll Interval**
   - Balance between freshness and rate limits
   - 3s is good for real-time, 10s+ for batch

2. **Use Custom RPC**
   - Public RPCs have strict rate limits
   - Paid RPCs support more requests/second

3. **Limit Tracked Accounts**
   - More accounts = more RPC calls
   - Use mint filtering to reduce noise

4. **Enable Supply Tracking Selectively**
   - Only enable if you need mint/burn alerts
   - Adds extra RPC calls

## 🎯 Advanced: Integration Ideas

### Database Storage

```go
// In tracker.go Process()
db.InsertMovement(ctx, &TokenMovement{
    Account: movement.Account,
    Amount: movement.Change,
    Timestamp: time.Now(),
})
```

### Webhook Alerts

```go
// In tracker.go checkAlerts()
if absChange > threshold {
    webhook.Send(WebhookPayload{
        Type: "significant_transfer",
        Account: movement.Account,
        Amount: movement.Change,
    })
}
```

### Prometheus Metrics

```yaml
metrics:
  backend: "prometheus"
```

Then add Prometheus exporter in main.go.

### Real-time Dashboard

Stream movements to WebSocket:

```go
wsServer.Broadcast(json.Marshal(movement))
```

## 📚 Code Structure Explained

### `decoder.go` - Data Decoding

Handles binary decoding of SPL Token data:
- Token account layout (165 bytes)
- Mint account layout (82 bytes)
- Support for Token Program & Token-2022

### `tracker.go` - Movement Tracking

Core tracking logic:
- Balance change detection
- Rich logging with emojis
- Alert triggering
- Human-readable formatting

### `mint_scanner.go` - Blockchain Scanning

Scans blockchain for token accounts:
- `getProgramAccounts` RPC calls
- Mint-based filtering
- Safety limits

### `main.go` - Pipeline Setup

Orchestrates the system:
- Config loading
- Pipeline building
- Graceful shutdown

## 📝 Related Examples

- [Basic Pipeline](../basic/) - Core pipeline concepts
- [Event Parser](../event-parser/) - Parse transaction events
- [Alerts](../alerts/) - Advanced alert system

## 🤝 Contributing

Found a bug or want to add a feature? Contributions welcome!

## 📄 License

MIT License - See root LICENSE file

---

**Made with ❤️ for the Solana ecosystem**
