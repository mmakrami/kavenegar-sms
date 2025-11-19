package kavenegar

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type Client struct {
	apiKey     string
	sender     string
	httpClient *http.Client
}

func NewClient(apiKey, sender string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	return &Client{
		apiKey:     apiKey,
		sender:     sender,
		httpClient: httpClient,
	}
}

// SendBulkSMS sends one SMS message to multiple receivers using Kavenegar API.
func (c *Client) SendBulkSMS(ctx context.Context, receivers []string, message string) error {
	if len(receivers) == 0 {
		return fmt.Errorf("no receivers provided")
	}

	form := url.Values{}
	form.Set("receptor", strings.Join(receivers, ","))
	form.Set("sender", c.sender)
	form.Set("message", message)

	endpoint := fmt.Sprintf("https://api.kavenegar.com/v1/%s/sms/send.json", c.apiKey)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("kavenegar returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
