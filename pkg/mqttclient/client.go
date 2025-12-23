package mqttclient

import (
	"fmt"
	"log/slog"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Config struct {
	BrokerURL string
	ClientID  string
	Username  string
	Password  string
}

type Client struct {
	client mqtt.Client
	logger *slog.Logger
}

func New(cfg Config, logger *slog.Logger) (*Client, error) {
	opts := mqtt.NewClientOptions().
		AddBroker(cfg.BrokerURL).
		SetClientID(cfg.ClientID).
		SetAutoReconnect(true).
		SetConnectRetry(true).
		SetConnectRetryInterval(5 * time.Second).
		SetMaxReconnectInterval(1 * time.Minute).
		SetKeepAlive(30 * time.Second).
		SetPingTimeout(10 * time.Second).
		SetCleanSession(false)

	if cfg.Username != "" {
		opts.SetUsername(cfg.Username)
	}
	if cfg.Password != "" {
		opts.SetPassword(cfg.Password)
	}

	opts.SetOnConnectHandler(func(c mqtt.Client) {
		logger.Info("connected to MQTT broker")
	})

	opts.SetConnectionLostHandler(func(c mqtt.Client, err error) {
		logger.Warn("connection lost to MQTT broker", "error", err)
	})

	opts.SetReconnectingHandler(func(c mqtt.Client, opts *mqtt.ClientOptions) {
		logger.Info("reconnecting to MQTT broker")
	})

	client := mqtt.NewClient(opts)

	token := client.Connect()
	if token.Wait() && token.Error() != nil {
		return nil, fmt.Errorf("failed to connect to MQTT broker: %w", token.Error())
	}

	return &Client{
		client: client,
		logger: logger,
	}, nil
}

func (c *Client) Subscribe(topic string, qos byte, handler mqtt.MessageHandler) error {
	token := c.client.Subscribe(topic, qos, handler)
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to subscribe to topic %s: %w", topic, token.Error())
	}
	c.logger.Info("subscribed to topic", "topic", topic)
	return nil
}

func (c *Client) Unsubscribe(topics ...string) error {
	token := c.client.Unsubscribe(topics...)
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to unsubscribe: %w", token.Error())
	}
	return nil
}

func (c *Client) Disconnect() {
	c.client.Disconnect(1000)
	c.logger.Info("disconnected from MQTT broker")
}

func (c *Client) IsConnected() bool {
	return c.client.IsConnected()
}
