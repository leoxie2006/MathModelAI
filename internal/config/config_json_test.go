package config

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestMCPConfigJSONUsesSnakeCaseFields(t *testing.T) {
	data, err := json.Marshal(MCPConfig{
		Enabled:         true,
		Host:            "127.0.0.1",
		Port:            8081,
		AuthHeader:      "X-MCP-Token",
		AuthHeaderValue: "secret",
	})
	if err != nil {
		t.Fatalf("marshal MCPConfig: %v", err)
	}
	body := string(data)

	for _, want := range []string{`"enabled":true`, `"host":"127.0.0.1"`, `"port":8081`, `"auth_header":"X-MCP-Token"`, `"auth_header_value":"secret"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("MCPConfig JSON = %s, missing %s", body, want)
		}
	}
	if strings.Contains(body, `"Enabled"`) || strings.Contains(body, `"Host"`) || strings.Contains(body, `"Port"`) {
		t.Fatalf("MCPConfig JSON should not expose Go field names: %s", body)
	}
}
