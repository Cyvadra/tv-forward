package handlers

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTradingViewAlertStruct(t *testing.T) {
	t.Run("JSON Marshaling", func(t *testing.T) {
		alert := TradingViewAlert{
			Strategy: "Test Strategy",
			Symbol:   "BTCUSDT",
			Action:   "buy",
			Price:    50000.0,
			Quantity: 0.1,
			Message:  "Test message",
			Exchange: "binance",
			Time:     "2024-01-01T00:00:00Z",
		}

		data, err := json.Marshal(alert)
		assert.NoError(t, err)
		assert.NotEmpty(t, data)

		var unmarshaled TradingViewAlert
		err = json.Unmarshal(data, &unmarshaled)
		assert.NoError(t, err)
		assert.Equal(t, alert.Strategy, unmarshaled.Strategy)
		assert.Equal(t, alert.Symbol, unmarshaled.Symbol)
		assert.Equal(t, alert.Action, unmarshaled.Action)
		assert.Equal(t, alert.Price, unmarshaled.Price)
		assert.Equal(t, alert.Quantity, unmarshaled.Quantity)
		assert.Equal(t, alert.Message, unmarshaled.Message)
		assert.Equal(t, alert.Exchange, unmarshaled.Exchange)
		assert.Equal(t, alert.Time, unmarshaled.Time)
	})

	t.Run("JSON Unmarshaling from string", func(t *testing.T) {
		jsonStr := `{
			"strategy": "RSI Strategy",
			"symbol": "ETHUSDT",
			"action": "sell",
			"price": 3000.0,
			"quantity": 0.5,
			"message": "RSI overbought signal",
			"exchange": "binance"
		}`

		var alert TradingViewAlert
		err := json.Unmarshal([]byte(jsonStr), &alert)
		assert.NoError(t, err)
		assert.Equal(t, "RSI Strategy", alert.Strategy)
		assert.Equal(t, "ETHUSDT", alert.Symbol)
		assert.Equal(t, "sell", alert.Action)
		assert.Equal(t, 3000.0, alert.Price)
		assert.Equal(t, 0.5, alert.Quantity)
		assert.Equal(t, "RSI overbought signal", alert.Message)
		assert.Equal(t, "binance", alert.Exchange)
	})

	t.Run("Invalid JSON handling", func(t *testing.T) {
		invalidJSON := `{"strategy": "test", "price": "not_a_number"}`

		var alert TradingViewAlert
		err := json.Unmarshal([]byte(invalidJSON), &alert)
		// This should error because "not_a_number" cannot be unmarshaled into float64
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot unmarshal string into Go struct field TradingViewAlert.price of type float64")
	})
}

func TestAlertHandlerCreation(t *testing.T) {
	t.Run("New Alert Handler", func(t *testing.T) {
		handler := NewAlertHandler()
		assert.NotNil(t, handler)
		assert.NotNil(t, handler.alertService)
		assert.NotNil(t, handler.forwardService)
		assert.NotNil(t, handler.tradingService)
	})
}
