package narodmon

import (
	"fmt"
	"net"
	"strings"
	"time"
)

// Client представляет клиент для отправки данных в Narodmon
type Client struct {
	server  string
	timeout time.Duration
}

// NewClient создаёт новый клиент Narodmon
func NewClient(server string, timeout int) *Client {
	return &Client{
		server:  server,
		timeout: time.Duration(timeout) * time.Second,
	}
}

// Sensor представляет данные одного датчика
type Sensor struct {
	ID    string  // Идентификатор датчика (например, "TEMP", "HUM")
	Value float64 // Значение
	Name  string  // Название датчика
}

// SendData отправляет данные в Narodmon по TCP
func (c *Client) SendData(mac, deviceName string, sensors []Sensor) error {
	// Формируем пакет данных
	packet := c.buildPacket(mac, deviceName, sensors)

	// Подключаемся по TCP
	conn, err := net.DialTimeout("tcp", c.server, c.timeout)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", c.server, err)
	}
	defer conn.Close()

	// Устанавливаем таймаут для записи
	if err := conn.SetWriteDeadline(time.Now().Add(c.timeout)); err != nil {
		return fmt.Errorf("failed to set write deadline: %w", err)
	}

	// Отправляем пакет
	if _, err := conn.Write([]byte(packet)); err != nil {
		return fmt.Errorf("failed to send data: %w", err)
	}

	// Читаем ответ
	if err := conn.SetReadDeadline(time.Now().Add(c.timeout)); err != nil {
		return fmt.Errorf("failed to set read deadline: %w", err)
	}

	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	response := string(buf[:n])
	if !strings.Contains(response, "OK") {
		return fmt.Errorf("unexpected response: %s", response)
	}

	return nil
}

// buildPacket формирует пакет данных в формате Narodmon
func (c *Client) buildPacket(mac, deviceName string, sensors []Sensor) string {
	var sb strings.Builder

	// Заголовок: #MAC#Название устройства
	sb.WriteString(fmt.Sprintf("#%s#%s\n", mac, deviceName))

	// Датчики: #ID#Value#Name
	for _, sensor := range sensors {
		sb.WriteString(fmt.Sprintf("#%s#%.2f#%s\n", sensor.ID, sensor.Value, sensor.Name))
	}

	// Завершающий маркер
	sb.WriteString("##\n")

	return sb.String()
}
