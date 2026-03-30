package mijnhost

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// --- NormalizeName / APIName ---

func TestNormalizeName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"example.com.", "example.com"},
		{"example.com", "example.com"},
		{"www.example.com.", "www.example.com"},
		{"", ""},
	}
	for _, tc := range tests {
		if got := NormalizeName(tc.input); got != tc.want {
			t.Errorf("NormalizeName(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestAPIName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"example.com", "example.com."},
		{"example.com.", "example.com."},
		{"www.example.com", "www.example.com."},
		{"", "."},
	}
	for _, tc := range tests {
		if got := APIName(tc.input); got != tc.want {
			t.Errorf("APIName(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestNormalizeRoundTrip(t *testing.T) {
	names := []string{"example.com", "www.example.com", "sub.domain.example.com"}
	for _, name := range names {
		if got := NormalizeName(APIName(name)); got != name {
			t.Errorf("round-trip failed for %q: got %q", name, got)
		}
	}
}

// --- helpers ---

func newTestServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *Client) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv, newClientWithBaseURL("test-api-key", srv.URL)
}

func assertHeader(t *testing.T, r *http.Request, key, want string) {
	t.Helper()
	if got := r.Header.Get(key); got != want {
		t.Errorf("header %q = %q, want %q", key, got, want)
	}
}

// --- GetDNSRecords ---

func TestGetDNSRecords(t *testing.T) {
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/domains/example.com/dns" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		assertHeader(t, r, "API-Key", "test-api-key")

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":             200,
			"status_description": "OK",
			"data": map[string]interface{}{
				"domain": "example.com",
				"records": []map[string]interface{}{
					{"type": "A", "name": "example.com.", "value": "1.2.3.4", "ttl": 3600},
					{"type": "MX", "name": "example.com.", "value": "10 mail.example.com.", "ttl": 3600},
				},
			},
		})
	})

	records, err := client.GetDNSRecords(context.Background(), "example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}

	// Trailing dots must be stripped.
	if records[0].Name != "example.com" {
		t.Errorf("record[0].Name = %q, want %q", records[0].Name, "example.com")
	}
	if records[0].Type != "A" || records[0].Value != "1.2.3.4" || records[0].TTL != 3600 {
		t.Errorf("unexpected record[0]: %+v", records[0])
	}
}

func TestGetDNSRecords_ErrorStatus(t *testing.T) {
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"status":401,"status_description":"No valid API key set"}`))
	})

	_, err := client.GetDNSRecords(context.Background(), "example.com")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- UpdateDNSRecords ---

func TestUpdateDNSRecords(t *testing.T) {
	var gotBody map[string]interface{}

	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("unexpected method: %s", r.Method)
		}
		assertHeader(t, r, "API-Key", "test-api-key")
		json.NewDecoder(r.Body).Decode(&gotBody)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": 200, "status_description": "OK"})
	})

	records := []DNSRecord{
		{Type: "A", Name: "example.com", Value: "1.2.3.4", TTL: 3600},
	}
	if err := client.UpdateDNSRecords(context.Background(), "example.com", records); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The API must receive the name with a trailing dot.
	recs := gotBody["records"].([]interface{})
	if len(recs) != 1 {
		t.Fatalf("expected 1 record in body, got %d", len(recs))
	}
	name := recs[0].(map[string]interface{})["name"].(string)
	if name != "example.com." {
		t.Errorf("API received name %q, want %q", name, "example.com.")
	}
}

// --- PatchDNSRecord ---

func TestPatchDNSRecord(t *testing.T) {
	var gotBody map[string]interface{}

	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("unexpected method: %s", r.Method)
		}
		assertHeader(t, r, "API-Key", "test-api-key")
		json.NewDecoder(r.Body).Decode(&gotBody)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": 200, "status_description": "OK"})
	})

	record := DNSRecord{Type: "A", Name: "www.example.com", Value: "5.6.7.8", TTL: 7200}
	if err := client.PatchDNSRecord(context.Background(), "example.com", record); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rec := gotBody["record"].(map[string]interface{})
	if rec["name"].(string) != "www.example.com." {
		t.Errorf("PATCH name = %q, want %q", rec["name"], "www.example.com.")
	}
	if rec["ttl"].(float64) != 7200 {
		t.Errorf("PATCH ttl = %v, want 7200", rec["ttl"])
	}
}

// --- ListDomains / GetDomain ---

func TestListDomains(t *testing.T) {
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/domains" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":             200,
			"status_description": "OK",
			"data": map[string]interface{}{
				"domains": []map[string]interface{}{
					{"id": 1, "domain": "example.com", "renewal_date": "2026-01-01", "status": "active", "tags": []string{"prod"}},
				},
			},
		})
	})

	domains, err := client.ListDomains(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(domains) != 1 || domains[0].Domain != "example.com" {
		t.Errorf("unexpected domains: %+v", domains)
	}
}

func TestGetDomain_Found(t *testing.T) {
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":             200,
			"status_description": "OK",
			"data": map[string]interface{}{
				"domains": []map[string]interface{}{
					{"id": 1, "domain": "example.com", "renewal_date": "2026-01-01", "status": "active", "tags": []string{}},
					{"id": 2, "domain": "other.com", "renewal_date": "2026-01-01", "status": "active", "tags": []string{}},
				},
			},
		})
	})

	d, err := client.GetDomain(context.Background(), "example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.Domain != "example.com" {
		t.Errorf("got domain %q, want %q", d.Domain, "example.com")
	}
}

func TestGetDomain_NotFound(t *testing.T) {
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":             200,
			"status_description": "OK",
			"data":               map[string]interface{}{"domains": []interface{}{}},
		})
	})

	_, err := client.GetDomain(context.Background(), "missing.com")
	if err == nil {
		t.Fatal("expected error for missing domain, got nil")
	}
}
