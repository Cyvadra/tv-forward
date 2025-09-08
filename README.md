# TV-Forward: TradingView Alert Forwarder

A Go-based webhook service that receives TradingView alerts and forwards them to multiple downstream platforms including Telegram, WeChat, DingTalk, and custom webhooks. It also supports automated trading on platforms like Bitget, Binance, and Derbit.

## Features

- **Multi-Platform Alert Forwarding**: Send TradingView alerts to Telegram, WeChat, DingTalk, and custom webhooks
- **Trading Integration**: Execute trades on Bitget, Binance, and Derbit platforms
- **Database Logging**: Store all alerts and trading signals in SQLite database
- **RESTful API**: Manage alerts and view trading signals via HTTP API
- **YAML Configuration**: Easy configuration management through YAML files
- **Gin Web Framework**: Fast and lightweight HTTP server
- **GORM ORM**: Database operations with automatic migrations

## Installation

### Prerequisites

- Go 1.22 or higher
- SQLite (included with the application)

### Build and Run

1. Clone the repository:
```bash
git clone https://github.com/Cyvadra/tv-forward.git
cd tv-forward
```

2. Build the application:
```bash
go build -o tv-forward ./cmd/main.go
```

3. Run the application:
```bash
./tv-forward
```

The application will create a default `config.yaml` file on first run.

## Configuration

Edit the `config.yaml` file to configure your endpoints and trading platforms:

```yaml
server:
  host: "localhost"
  port: "9006"

database:
  driver: "sqlite"
  dsn: "tv-forward.db"

endpoints:
  - name: "Telegram Bot"
    type: "telegram"
    url: ""
    token: "YOUR_TELEGRAM_BOT_TOKEN"
    chat_id: "YOUR_CHAT_ID"
    is_active: true

  - name: "WeChat Bot"
    type: "wechat"
    url: "YOUR_WECHAT_WEBHOOK_URL"
    token: ""
    chat_id: ""
    is_active: false

  - name: "DingTalk Bot"
    type: "dingtalk"
    url: "YOUR_DINGTALK_WEBHOOK_URL"
    token: ""
    chat_id: ""
    is_active: false

trading:
  bitget:
    api_key: "YOUR_BITGET_API_KEY"
    secret_key: "YOUR_BITGET_SECRET_KEY"
    passphrase: "YOUR_BITGET_PASSPHRASE"
    is_active: false

  binance:
    api_key: "YOUR_BINANCE_API_KEY"
    secret_key: "YOUR_BINANCE_SECRET_KEY"
    is_active: false

  derbit:
    api_key: "YOUR_DERBIT_API_KEY"
    secret_key: "YOUR_DERBIT_SECRET_KEY"
    is_active: false
```

## API Endpoints

### TradingView Webhook
- **POST** `/api/v1/webhook/tradingview`
- Receives TradingView alerts
- Expected JSON format:
```json
{
  "strategy": "My Strategy",
  "symbol": "BTCUSDT",
  "action": "buy",
  "price": 50000.0,
  "quantity": 0.1,
  "message": "Buy signal triggered"
}
```

### Alert Management
- **GET** `/api/v1/alerts` - List all alerts with pagination
- **GET** `/api/v1/alerts/:id` - Get specific alert by ID
- **GET** `/api/v1/alerts/:alertId/signals` - Get trading signals for an alert

### Health Check
- **GET** `/health` - Service health status

## Setting Up TradingView Alerts

1. In TradingView, create a new alert
2. Set the alert action to "Webhook URL"
3. Use the URL: `http://your-server:9006/api/v1/webhook/tradingview`
4. Configure the alert message as JSON with the required fields

## Supported Platforms

### Alert Platforms
- **Telegram**: Send alerts to Telegram channels/groups
- **WeChat**: Enterprise WeChat bot integration
- **DingTalk**: DingTalk bot integration
- **Custom Webhooks**: Forward to any HTTP endpoint

### Trading Platforms
- **Bitget**: Spot and futures trading
- **Binance**: Spot and futures trading
- **Derbit**: Options and futures trading

## Database Schema

The application uses SQLite with the following tables:

- **alerts**: Stores all incoming TradingView alerts
- **trading_signals**: Records trading executions
- **downstream_endpoints**: Configuration for alert forwarding

## Development

### Project Structure
```
tv-forward/
├── cmd/
│   └── main.go              # Application entry point
├── internal/
│   ├── config/              # Configuration management
│   ├── database/            # Database initialization
│   ├── handlers/            # HTTP request handlers
│   ├── models/              # Database models
│   ├── routes/              # Route definitions
│   └── services/            # Business logic services
├── config.yaml              # Configuration file
├── go.mod                   # Go module file
└── README.md               # This file
```

### Running Tests
```bash
go test ./...
```

### Building for Production
```bash
go build -ldflags="-s -w" -o tv-forward ./cmd/main.go
```

## Security Considerations

- Store API keys and tokens securely
- Use HTTPS in production
- Implement rate limiting for webhook endpoints
- Regularly rotate API keys
- Monitor trading activities

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Support

For issues and questions, please open an issue on GitHub.
