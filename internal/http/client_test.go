package http_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	httpclient "github.com/parikhrahil/httpgo/internal/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecuteHTTPRequest_RoundTrip(t *testing.T) {
	var (
		gotMethod string
		gotHeader string
		gotBody   string
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotHeader = r.Header.Get("X-Probe")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)

		w.Header().Set("X-Reply", "pong")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	req, err := http.NewRequest(http.MethodPost, srv.URL+"/v1/things", strings.NewReader("payload"))
	require.NoError(t, err)
	req.Header.Set("X-Probe", "value")

	res, body, err := httpclient.ExecuteHTTPRequest(req, 0)
	require.NoError(t, err)

	assert.Equal(t, http.StatusCreated, res.StatusCode)
	assert.Equal(t, "pong", res.Header.Get("X-Reply"))
	assert.Equal(t, `{"ok":true}`, string(body))

	assert.Equal(t, http.MethodPost, gotMethod)
	assert.Equal(t, "value", gotHeader)
	assert.Equal(t, "payload", gotBody)
}

func TestExecuteHTTPRequest_ClearsRequestURI(t *testing.T) {
	// http.Client.Do panics with "Request.RequestURI must be empty" when the
	// field is set — so this also covers requests built via http.ReadRequest
	// (which sets it). ExecuteHTTPRequest must clear it before dispatching.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/path", nil)
	require.NoError(t, err)
	req.RequestURI = "/path"

	require.NotPanics(t, func() {
		res, _, err := httpclient.ExecuteHTTPRequest(req, 0)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, res.StatusCode)
	})
}

func TestExecuteHTTPRequest_PerRequestTimeoutFires(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(2 * time.Second):
			w.WriteHeader(http.StatusOK)
		case <-r.Context().Done():
		}
	}))
	defer srv.Close()

	req, err := http.NewRequest(http.MethodGet, srv.URL, nil)
	require.NoError(t, err)

	start := time.Now()
	_, _, err = httpclient.ExecuteHTTPRequest(req, 50*time.Millisecond)
	elapsed := time.Since(start)

	require.Error(t, err)
	assert.Less(t, elapsed, time.Second,
		"timeout should fire well before the handler's 2s sleep completes")
	// net/http reports the cancel cause as context.DeadlineExceeded but wraps
	// it inside a *url.Error, so use errors.Is for the comparison.
	assert.True(t, errors.Is(err, context.DeadlineExceeded),
		"expected DeadlineExceeded, got %v", err)
}

func TestExecuteHTTPRequest_TimeoutZeroSkipsPerRequestDeadline(t *testing.T) {
	// A 250ms server pause must succeed when no per-request timeout is set
	// (the client-wide 30s timeout is the only deadline in play).
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(250 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	req, err := http.NewRequest(http.MethodGet, srv.URL, nil)
	require.NoError(t, err)

	res, _, err := httpclient.ExecuteHTTPRequest(req, 0)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode)
}

func TestExecuteHTTPRequest_NetworkFailureIsWrapped(t *testing.T) {
	// 127.0.0.1:1 is the standard "nothing listens here" address.
	req, err := http.NewRequest(http.MethodGet, "http://127.0.0.1:1/", nil)
	require.NoError(t, err)

	_, _, err = httpclient.ExecuteHTTPRequest(req, 200*time.Millisecond)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "network request failed")
}
