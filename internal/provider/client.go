package provider

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Client is a minimal HTTP client for talking to the Seq HTTP API.
//
// Authentication uses the X-Seq-ApiKey header (recommended by Seq).
// Ref: https://datalust.co/docs/using-the-http-api
type Client struct {
	baseURL *url.URL
	apiKey  string
	http    *http.Client
}

func NewClientFromConfig(ctx context.Context, cfg SeqProviderModel) (*Client, diag.Diagnostics) {
	var diags diag.Diagnostics

	serverURL := firstNonEmpty(
		stringValue(cfg.ServerURL),
		os.Getenv("SEQ_SERVER_URL"),
	)
	if serverURL == "" {
		diags.AddError(
			"Missing Seq server_url",
			"Configure the provider with server_url or set SEQ_SERVER_URL.",
		)
		return nil, diags
	}

	parsed, err := url.Parse(serverURL)
	if err != nil {
		diags.AddError("Invalid server_url", err.Error())
		return nil, diags
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		diags.AddError("Invalid server_url", "server_url must include scheme and host, e.g. http://localhost:5342")
		return nil, diags
	}

	apiKey := firstNonEmpty(
		stringValue(cfg.APIKey),
		os.Getenv("SEQ_API_KEY"),
	)

	insecureSkipVerify := boolValue(cfg.InsecureSkipVerify)
	if env := os.Getenv("SEQ_INSECURE_SKIP_VERIFY"); env != "" {
		if v, err := strconv.ParseBool(env); err == nil {
			insecureSkipVerify = v
		}
	}

	timeoutSeconds := int64Value(cfg.TimeoutSeconds)
	if timeoutSeconds == 0 {
		timeoutSeconds = 30
	}
	if env := os.Getenv("SEQ_TIMEOUT_SECONDS"); env != "" {
		if v, err := strconv.ParseInt(env, 10, 64); err == nil {
			timeoutSeconds = v
		}
	}

	httpClient := &http.Client{
		Timeout: time.Duration(timeoutSeconds) * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: insecureSkipVerify},
		},
	}

	c := &Client{baseURL: parsed, apiKey: apiKey, http: httpClient}

	// Best-effort connectivity check.
	if err := c.Ping(ctx); err != nil {
		tflog.Warn(ctx, "Seq provider configured, but /health check failed", map[string]any{"error": err.Error()})
	}

	return c, diags
}

func (c *Client) Ping(ctx context.Context) error {
	var out map[string]any
	return c.doJSON(ctx, http.MethodGet, "/health", nil, &out)
}

// doJSON performs an HTTP request with JSON body/response.
func (c *Client) doJSON(ctx context.Context, method, path string, body any, out any) error {
	fullURL, err := c.baseURL.Parse(strings.TrimPrefix(path, "/"))
	if err != nil {
		return err
	}

	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL.String(), bodyReader)
	if err != nil {
		return err
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.apiKey != "" {
		req.Header.Set("X-Seq-ApiKey", c.apiKey)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		msg := strings.TrimSpace(string(data))
		if msg == "" {
			msg = resp.Status
		}
		return &HTTPError{StatusCode: resp.StatusCode, Message: msg}
	}

	if out == nil {
		return nil
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return nil
	}

	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("decode JSON response: %w", err)
	}

	return nil
}

// HTTPError wraps non-2xx responses.
type HTTPError struct {
	StatusCode int
	Message    string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("seq api returned %d: %s", e.StatusCode, e.Message)
}

func firstNonEmpty(vs ...string) string {
	for _, v := range vs {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func stringValue(v types.String) string {
	if v.IsNull() || v.IsUnknown() {
		return ""
	}
	return v.ValueString()
}

func boolValue(v types.Bool) bool {
	if v.IsNull() || v.IsUnknown() {
		return false
	}
	return v.ValueBool()
}

func int64Value(v types.Int64) int64 {
	if v.IsNull() || v.IsUnknown() {
		return 0
	}
	return v.ValueInt64()
}
