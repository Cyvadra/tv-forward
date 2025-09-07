# TV-Forward Features

## Overview
TV-Forward now supports advanced TradingView signal processing with user isolation, position tracking, and multi-exchange support.

## Key Features

### 1. API_SEC Based User Isolation
- Each `api_sec` represents a separate user account
- Users are automatically created when first signal is received
- Complete isolation of trading data, positions, and signals per user

### 2. TradingView Signal Format Support
The system now supports the full TradingView signal format as shown in `docs/tvcbot.json`:

```json
{
  "ticker": "ETHUSDT.P",
  "ex": "BITGET",
  "position_size": "4.8193",
  "action": "buy",
  "market_position": "long",
  "market_position_size": "4.8193",
  "prev_market_position": "flat",
  "prev_market_position_size": "0",
  "exchange": "bitget",
  "lever": 22,
  "td_mode": "isolated",
  "symbol": "ETHUSDT_UMCBL",
  "api_sec": "asdfasdfasdfasdf"
}
```

### 3. Position Tracking & Validation
- Validates `prev_market_position_size` against current positions
- Prevents duplicate trades and network errors
- Automatic position updates after successful trades
- Position history tracking

### 4. Multi-Exchange Support
- **Binance**: Spot and futures trading
- **Bitget**: Futures trading with passphrase support
- **OKX**: Futures trading with passphrase support
- **Derbit**: Legacy support maintained

### 5. User Configuration Management
Users are configured in `users.yaml`:

```yaml
users:
  - api_sec: "user_api_key_here"
    name: "User Name"
    is_active: true
    credentials:
      - exchange: "binance"
        api_key: "YOUR_API_KEY"
        secret_key: "YOUR_SECRET_KEY"
        is_active: true
      - exchange: "bitget"
        api_key: "YOUR_API_KEY"
        secret_key: "YOUR_SECRET_KEY"
        passphrase: "YOUR_PASSPHRASE"
        is_active: true
```

### 6. Signal Storage & History
- All received signals are stored with full payload
- Trading execution records (successful and failed)
- Position change history
- User-specific signal retrieval

## API Endpoints

### Webhook Endpoint
- `POST /api/v1/webhook/tradingview` - Receives TradingView signals

### User Management
- `GET /api/v1/users/{api_sec}/signals` - Get user's trading signals
- `GET /api/v1/users/{api_sec}/positions` - Get user's current positions

### Alert Management (Legacy)
- `GET /api/v1/alerts` - Get all alerts
- `GET /api/v1/alerts/{id}` - Get specific alert
- `GET /api/v1/alerts/{id}/signals` - Get trading signals for alert

## Signal Processing Flow

1. **Signal Reception**: TradingView sends signal to webhook
2. **User Identification**: System identifies user by `api_sec`
3. **Position Validation**: Checks `prev_market_position_size` against current position
4. **Exchange Selection**: Uses signal's `exchange` field to determine target exchange
5. **Credential Retrieval**: Gets user's credentials for the specified exchange
6. **Order Execution**: Places order on exchange (placeholder implementation)
7. **Position Update**: Updates user's position after successful trade
8. **Record Storage**: Saves signal and execution records

## Configuration Files

### config.yaml
Main application configuration including server settings and exchange configurations.

### users.yaml
User-specific configuration including API credentials for each exchange.

## Database Schema

### New Tables
- `users`: User accounts identified by api_sec
- `user_credentials`: Exchange credentials per user
- `positions`: Current trading positions per user
- `trading_signals`: Enhanced signal storage with user association

### Enhanced Tables
- `alerts`: Legacy alert support maintained
- `trading_signals`: Now includes user isolation and position tracking

## Testing

Use the provided `test_signal.json` to test the new signal format:

```bash
curl -X POST http://localhost:8080/api/v1/webhook/tradingview \
  -H "Content-Type: application/json" \
  -d @test_signal.json
```

## Migration Notes

- Existing alerts continue to work with legacy format
- New TradingView signals are automatically detected by presence of `api_sec` field
- Database migration is automatic on startup
- User configuration is loaded from `users.yaml` on startup
