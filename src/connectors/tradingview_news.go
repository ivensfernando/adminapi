package connectors

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strategyexecutor/src/model"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

type ClientTV struct {
	httpClient *http.Client
}

func NewClientTV(httpClient *http.Client) *ClientTV {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	return &ClientTV{httpClient: httpClient}
}

func (c *ClientTV) FetchImportantEvents(
	ctx context.Context,
	fromUTC time.Time,
	toUTC time.Time,
	countries []string,
) ([]model.Event, error) {
	baseURL := "https://economic-calendar.tradingview.com/events"

	q := url.Values{}
	q.Set("from", fromUTC.UTC().Format("2006-01-02T15:04:05.000Z"))
	q.Set("to", toUTC.UTC().Format("2006-01-02T15:04:05.000Z"))
	q.Set("countries", strings.Join(countries, ","))

	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse base url: %w", err)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}

	req.Header.Set("accept", "application/json")
	req.Header.Set("accept-language", "en-GB,en;q=0.9")
	req.Header.Set("origin", "https://www.tradingview.com")
	req.Header.Set("referer", "https://www.tradingview.com/")
	req.Header.Set("user-agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Safari/537.36")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
		return nil, fmt.Errorf("unexpected status %d. body: %s", resp.StatusCode, string(b))
	}

	var decoded model.EventsResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, fmt.Errorf("decode json: %w", err)
	}
	if decoded.Status != "ok" && decoded.Status != "" {
		return nil, fmt.Errorf("unexpected status field: %q", decoded.Status)
	}

	out := make([]model.Event, 0, len(decoded.Result))

	logrus.Infof("Fetched %d events", len(decoded.Result))

	for _, ev := range decoded.Result {
		if ev.Importance == 1 {
			out = append(out, ev)
		}
	}
	return out, nil
}
