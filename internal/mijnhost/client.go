package mijnhost

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const defaultBaseURL = "https://mijn.host/api/v2"

// Client is the mijn.host API client.
type Client struct {
	apiKey     string
	httpClient *http.Client
	baseURL    string
}

// NewClient creates a new mijn.host API client.
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey:     apiKey,
		httpClient: &http.Client{},
		baseURL:    defaultBaseURL,
	}
}

// newClientWithBaseURL creates a client with a custom base URL (used in tests).
func newClientWithBaseURL(apiKey, baseURL string) *Client {
	return &Client{
		apiKey:     apiKey,
		httpClient: &http.Client{},
		baseURL:    baseURL,
	}
}

// DNSRecord represents a single DNS record.
type DNSRecord struct {
	Type  string `json:"type"`
	Name  string `json:"name"`
	Value string `json:"value"`
	TTL   int64  `json:"ttl"`
}

type dnsResponse struct {
	Status            int    `json:"status"`
	StatusDescription string `json:"status_description"`
	Data              struct {
		Domain  string      `json:"domain"`
		Records []DNSRecord `json:"records"`
	} `json:"data"`
}

// Domain represents a mijn.host domain.
type Domain struct {
	ID          int      `json:"id"`
	Domain      string   `json:"domain"`
	RenewalDate string   `json:"renewal_date"`
	Status      string   `json:"status"`
	Tags        []string `json:"tags"`
}

type domainsResponse struct {
	Status            int    `json:"status"`
	StatusDescription string `json:"status_description"`
	Data              struct {
		Domains []Domain `json:"domains"`
	} `json:"data"`
}

type domainResponse struct {
	Status            int    `json:"status"`
	StatusDescription string `json:"status_description"`
	Data              Domain `json:"data"`
}

func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshaling request body: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("API-Key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}

	return resp, nil
}

// GetDNSRecords retrieves all DNS records for the given domain.
func (c *Client) GetDNSRecords(ctx context.Context, domain string) ([]DNSRecord, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/domains/"+domain+"/dns", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var result dnsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	// Normalize names: remove trailing dots returned by the API.
	for i := range result.Data.Records {
		result.Data.Records[i].Name = NormalizeName(result.Data.Records[i].Name)
	}

	return result.Data.Records, nil
}

// UpdateDNSRecords replaces ALL DNS records for the domain with the provided set.
// This is a destructive full-replace operation.
func (c *Client) UpdateDNSRecords(ctx context.Context, domain string, records []DNSRecord) error {
	// The API expects names with trailing dots.
	apiRecords := make([]DNSRecord, len(records))
	for i, r := range records {
		apiRecords[i] = r
		apiRecords[i].Name = APIName(r.Name)
	}

	payload := map[string]interface{}{
		"records": apiRecords,
	}

	resp, err := c.doRequest(ctx, http.MethodPut, "/domains/"+domain+"/dns", payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// PatchDNSRecord updates a single DNS record without affecting others.
func (c *Client) PatchDNSRecord(ctx context.Context, domain string, record DNSRecord) error {
	apiRecord := record
	apiRecord.Name = APIName(record.Name)

	payload := map[string]interface{}{
		"record": apiRecord,
	}

	resp, err := c.doRequest(ctx, http.MethodPatch, "/domains/"+domain+"/dns", payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// ListDomains returns all domains in the account.
func (c *Client) ListDomains(ctx context.Context) ([]Domain, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/domains", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var result domainsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return result.Data.Domains, nil
}

// GetDomain returns a single domain by name.
func (c *Client) GetDomain(ctx context.Context, domain string) (*Domain, error) {
	domains, err := c.ListDomains(ctx)
	if err != nil {
		return nil, err
	}
	for _, d := range domains {
		if d.Domain == domain {
			return &d, nil
		}
	}
	return nil, fmt.Errorf("domain %q not found", domain)
}

// NormalizeName removes trailing dots from DNS record names for consistent storage in state.
func NormalizeName(name string) string {
	return strings.TrimSuffix(name, ".")
}

// APIName adds a trailing dot to a DNS record name as required by the mijn.host API.
func APIName(name string) string {
	if strings.HasSuffix(name, ".") {
		return name
	}
	return name + "."
}
