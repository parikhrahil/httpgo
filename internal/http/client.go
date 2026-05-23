package http

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

const defaultTimeout time.Duration = 30 * time.Second

var client *http.Client

func init() {
	client = &http.Client{
		Timeout: defaultTimeout,
		Transport: &http.Transport{
			// MaxIdleConns controls the total connection pool size across all hosts
			MaxIdleConns: 100,
			// MaxIdleConnsPerHost controls pool size for concurrent threads hitting the same host
			MaxIdleConnsPerHost: 20,
			// How long an idle connection stays alive in the pool before closing
			IdleConnTimeout: 90 * time.Second,
			DialContext: (&net.Dialer{
				Timeout:   10 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout: 10 * time.Second,
		},
	}
}

func ExecuteHTTPRequest(req *http.Request, timeout time.Duration) (*http.Response, []byte, error) {
	// Clear out RequestURI for outgoing client requests to avoid runtime panics
	req.RequestURI = ""

	ctx := req.Context()
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	res, err := client.Do(req.WithContext(ctx))
	if err != nil {
		return nil, nil, fmt.Errorf("network request failed: %w", err)
	}
	defer res.Body.Close()

	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read response body: %w", err)
	}
	return res, bodyBytes, nil
}
