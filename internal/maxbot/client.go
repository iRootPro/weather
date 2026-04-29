package maxbot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const defaultAPIBaseURL = "https://platform-api.max.ru"

type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

func NewClient(token string, timeout time.Duration) *Client {
	return &Client{baseURL: defaultAPIBaseURL, token: token, httpClient: &http.Client{Timeout: timeout}}
}

func (c *Client) GetMe(ctx context.Context) (*BotInfo, error) {
	var bot BotInfo
	if err := c.do(ctx, http.MethodGet, "/me", nil, nil, &bot); err != nil {
		return nil, err
	}
	return &bot, nil
}

func (c *Client) GetUpdates(ctx context.Context, marker *int64, limit, timeout int, types []string) (*UpdateList, error) {
	q := url.Values{}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	if timeout >= 0 {
		q.Set("timeout", strconv.Itoa(timeout))
	}
	if marker != nil {
		q.Set("marker", strconv.FormatInt(*marker, 10))
	}
	if len(types) > 0 {
		q.Set("types", strings.Join(types, ","))
	}
	var updates UpdateList
	if err := c.do(ctx, http.MethodGet, "/updates", q, nil, &updates); err != nil {
		return nil, err
	}
	return &updates, nil
}

func (c *Client) SendMessageToUser(ctx context.Context, userID int64, body NewMessageBody) error {
	q := url.Values{}
	q.Set("user_id", strconv.FormatInt(userID, 10))
	return c.do(ctx, http.MethodPost, "/messages", q, body, nil)
}

func (c *Client) AnswerCallback(ctx context.Context, callbackID, notification string) error {
	q := url.Values{}
	q.Set("callback_id", callbackID)
	return c.do(ctx, http.MethodPost, "/answers", q, CallbackAnswer{Notification: notification}, nil)
}

func (c *Client) do(ctx context.Context, method, path string, q url.Values, in interface{}, out interface{}) error {
	u, err := url.Parse(c.baseURL + path)
	if err != nil {
		return err
	}
	if q != nil {
		u.RawQuery = q.Encode()
	}

	var body io.Reader
	if in != nil {
		buf, err := json.Marshal(in)
		if err != nil {
			return err
		}
		body = bytes.NewReader(buf)
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", c.token)
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("max api %s %s failed: status=%d body=%s", method, path, resp.StatusCode, string(data))
	}
	if out == nil || len(data) == 0 {
		return nil
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("decode max api response: %w", err)
	}
	return nil
}
