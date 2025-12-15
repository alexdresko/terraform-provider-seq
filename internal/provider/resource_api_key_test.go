package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestClientAddsAPIKeyHeader(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("X-Seq-ApiKey")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer srv.Close()

	c := &Client{baseURL: mustParseURL(srv.URL), apiKey: "abc", http: srv.Client()}
	if err := c.Ping(context.Background()); err != nil {
		t.Fatalf("Ping() error: %v", err)
	}
	if got != "abc" {
		t.Fatalf("expected X-Seq-ApiKey header to be set, got %q", got)
	}
}

func TestAPIKeyRequestBody(t *testing.T) {
	m := APIKeyModel{
		Title:       types.StringValue("x"),
		OwnerID:     types.StringValue("owner"),
		Permissions: types.SetValueMust(types.StringType, []attr.Value{types.StringValue("Read"), types.StringValue("Write")}),
	}
	body, diags := apiKeyRequestBody(context.Background(), m)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics")
	}
	if body["Title"].(string) != "x" {
		t.Fatalf("title mismatch")
	}
}
