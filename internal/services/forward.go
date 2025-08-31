package services

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Cyvadra/tv-forward/internal/config"
	"github.com/Cyvadra/tv-forward/internal/models"
	"github.com/go-resty/resty/v2"
)

// ForwardService handles forwarding alerts to downstream endpoints
type ForwardService struct {
	client *resty.Client
	config *config.Config
}

// NewForwardService creates a new forward service
func NewForwardService() *ForwardService {
	return &ForwardService{
		client: resty.New().SetTimeout(10 * time.Second),
		config: nil, // Will be set later
	}
}

// SetConfig sets the configuration for the forward service
func (s *ForwardService) SetConfig(cfg *config.Config) {
	s.config = cfg
}

// ForwardAlert forwards an alert to all configured downstream endpoints
func (s *ForwardService) ForwardAlert(alert *models.Alert) error {
	if s.config == nil {
		return fmt.Errorf("configuration not set")
	}

	for _, endpoint := range s.config.Endpoints {
		if !endpoint.IsActive {
			continue
		}

		go func(ep config.EndpointConfig) {
			if err := s.forwardToEndpoint(alert, ep); err != nil {
				log.Printf("Failed to forward to %s (%s): %v", ep.Name, ep.Type, err)
			}
		}(endpoint)
	}

	return nil
}

// forwardToEndpoint forwards an alert to a specific endpoint
func (s *ForwardService) forwardToEndpoint(alert *models.Alert, endpoint config.EndpointConfig) error {
	switch endpoint.Type {
	case "telegram":
		return s.forwardToTelegram(alert, endpoint)
	case "wechat":
		return s.forwardToWeChat(alert, endpoint)
	case "dingtalk":
		return s.forwardToDingTalk(alert, endpoint)
	case "webhook":
		return s.forwardToWebhook(alert, endpoint)
	default:
		return fmt.Errorf("unsupported endpoint type: %s", endpoint.Type)
	}
}

// forwardToTelegram forwards an alert to Telegram
func (s *ForwardService) forwardToTelegram(alert *models.Alert, endpoint config.EndpointConfig) error {
	message := s.formatRawMessage(alert)

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", endpoint.Token)
	payload := map[string]interface{}{
		"chat_id":    endpoint.ChatID,
		"text":       message,
		"parse_mode": "HTML",
	}

	resp, err := s.client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(payload).
		Post(url)

	if err != nil {
		return fmt.Errorf("telegram API request failed: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("telegram API returned status %d: %s", resp.StatusCode(), resp.String())
	}

	return nil
}

// forwardToWeChat forwards an alert to WeChat (Enterprise WeChat)
func (s *ForwardService) forwardToWeChat(alert *models.Alert, endpoint config.EndpointConfig) error {
	message := s.formatRawMessage(alert)

	payload := map[string]interface{}{
		"msgtype": "text",
		"text": map[string]string{
			"content": message,
		},
	}

	resp, err := s.client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(payload).
		Post(endpoint.URL)

	if err != nil {
		return fmt.Errorf("wechat API request failed: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("wechat API returned status %d: %s", resp.StatusCode(), resp.String())
	}

	return nil
}

// forwardToDingTalk forwards an alert to DingTalk
func (s *ForwardService) forwardToDingTalk(alert *models.Alert, endpoint config.EndpointConfig) error {
	message := s.formatRawMessage(alert)

	payload := map[string]interface{}{
		"msgtype": "text",
		"text": map[string]string{
			"content": message,
		},
	}

	resp, err := s.client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(payload).
		Post(endpoint.URL)

	if err != nil {
		return fmt.Errorf("dingtalk API request failed: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("dingtalk API returned status %d: %s", resp.StatusCode(), resp.String())
	}

	return nil
}

// forwardToWebhook forwards an alert to a generic webhook
func (s *ForwardService) forwardToWebhook(alert *models.Alert, endpoint config.EndpointConfig) error {
	resp, err := s.client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(alert).
		Post(endpoint.URL)

	if err != nil {
		return fmt.Errorf("webhook request failed: %w", err)
	}

	if resp.StatusCode() < 200 || resp.StatusCode() >= 300 {
		return fmt.Errorf("webhook returned status %d: %s", resp.StatusCode(), resp.String())
	}

	return nil
}

// formatTelegramMessage formats the alert message for Telegram
func (s *ForwardService) formatTelegramMessage(alert *models.Alert) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🚨 <b>Trading Alert</b>\n\n"))
	sb.WriteString(fmt.Sprintf("📊 <b>Strategy:</b> %s\n", alert.Strategy))
	sb.WriteString(fmt.Sprintf("💱 <b>Symbol:</b> %s\n", alert.Symbol))
	sb.WriteString(fmt.Sprintf("⚡ <b>Action:</b> %s\n", strings.ToUpper(alert.Action)))
	sb.WriteString(fmt.Sprintf("💰 <b>Price:</b> %.8f\n", alert.Price))
	if alert.Quantity > 0 {
		sb.WriteString(fmt.Sprintf("📈 <b>Quantity:</b> %.8f\n", alert.Quantity))
	}
	if alert.Message != "" {
		sb.WriteString(fmt.Sprintf("💬 <b>Message:</b> %s\n", alert.Message))
	}
	sb.WriteString(fmt.Sprintf("⏰ <b>Time:</b> %s", alert.CreatedAt.Format("2006-01-02 15:04:05")))
	return sb.String()
}

// formatWeChatMessage formats the alert message for WeChat
func (s *ForwardService) formatWeChatMessage(alert *models.Alert) string {
	var sb strings.Builder
	sb.WriteString("🚨 Trading Alert\n\n")
	sb.WriteString(fmt.Sprintf("📊 Strategy: %s\n", alert.Strategy))
	sb.WriteString(fmt.Sprintf("💱 Symbol: %s\n", alert.Symbol))
	sb.WriteString(fmt.Sprintf("⚡ Action: %s\n", strings.ToUpper(alert.Action)))
	sb.WriteString(fmt.Sprintf("💰 Price: %.8f\n", alert.Price))
	if alert.Quantity > 0 {
		sb.WriteString(fmt.Sprintf("📈 Quantity: %.8f\n", alert.Quantity))
	}
	if alert.Message != "" {
		sb.WriteString(fmt.Sprintf("💬 Message: %s\n", alert.Message))
	}
	sb.WriteString(fmt.Sprintf("⏰ Time: %s", alert.CreatedAt.Format("2006-01-02 15:04:05")))
	return sb.String()
}

// formatDingTalkMessage formats the alert message for DingTalk
func (s *ForwardService) formatDingTalkMessage(alert *models.Alert) string {
	var sb strings.Builder
	sb.WriteString("🚨 Trading Alert\n\n")
	sb.WriteString(fmt.Sprintf("📊 Strategy: %s\n", alert.Strategy))
	sb.WriteString(fmt.Sprintf("💱 Symbol: %s\n", alert.Symbol))
	sb.WriteString(fmt.Sprintf("⚡ Action: %s\n", strings.ToUpper(alert.Action)))
	sb.WriteString(fmt.Sprintf("💰 Price: %.8f\n", alert.Price))
	if alert.Quantity > 0 {
		sb.WriteString(fmt.Sprintf("📈 Quantity: %.8f\n", alert.Quantity))
	}
	if alert.Message != "" {
		sb.WriteString(fmt.Sprintf("💬 Message: %s\n", alert.Message))
	}
	sb.WriteString(fmt.Sprintf("⏰ Time: %s", alert.CreatedAt.Format("2006-01-02 15:04:05")))
	return sb.String()
}

// formatRawMessage formats the alert message for raw message
func (s *ForwardService) formatRawMessage(alert *models.Alert) string {
	return alert.Message
}
