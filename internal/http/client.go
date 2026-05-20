package http

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

var client *http.Client

func init() {
	client = &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			// MaxIdleConns controls the total connection pool size across all hosts
			MaxIdleConns:        100,
			// MaxIdleConnsPerHost controls pool size for concurrent threads hitting the same host
			MaxIdleConnsPerHost: 20,
			// How long an idle connection stays alive in the pool before closing
			IdleConnTimeout:     90 * time.Second,
			DialContext: (&net.Dialer{
				Timeout:   10 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout: 10 * time.Second,
		},
	}
}

func ExecuteHTTPRequest(req *http.Request) (*http.Response, []byte, error) {
	// Clear out RequestURI for outgoing client requests to avoid runtime panics
	req.RequestURI = ""

	res, err := client.Do(req)
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