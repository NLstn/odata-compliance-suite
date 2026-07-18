package framework

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// TestSuite represents a collection of compliance tests
type TestSuite struct {
	Name        string
	Description string
	SpecURL     string
	Tests       []Test
	Results     *TestResults
	ServerURL   string
	Debug       bool
	Verbose     bool
	Quiet       bool
	Client      *http.Client
	// Out is the writer for all human-readable progress and summary output.
	// Defaults to os.Stdout; callers may redirect it to os.Stderr when structured
	// report output owns stdout.
	Out io.Writer

	// Capabilities holds the parsed service capability profile; nil disables capability gating.
	Capabilities *CapabilityProfile
	// Strict disables capability-based skipping so all tests run regardless of declared restrictions.
	Strict bool
	// RequiredCapabilities lists capabilities this suite depends on. When Strict is false and the
	// service declares any of them unsupported, every test in the suite is recorded as skipped.
	RequiredCapabilities []RequiredCapability
}

// Test represents a single test case
type Test struct {
	Name                 string
	Description          string
	RequiredCapabilities []RequiredCapability
	Fn                   func(*TestContext) error
}

// TestResults tracks the results of test execution
type TestResults struct {
	Total   int
	Passed  int
	Failed  int
	Skipped int
	Details []TestDetail
}

// AllSkipped reports whether every test in the suite was skipped (typically
// because the service declared a required capability unsupported). Run()
// returns nil in this case exactly as it does for a fully-passing suite, so
// callers that need to distinguish "verified compliant" from "nothing was
// actually checked" must consult this explicitly rather than trusting a nil
// error or a "PASSING" status alone.
func (r *TestResults) AllSkipped() bool {
	return r.Total > 0 && r.Skipped == r.Total
}

// TestDetail contains information about a single test result
type TestDetail struct {
	Name   string
	Status TestStatus
	Error  string
}

// TestStatus represents the status of a test
type TestStatus int

const (
	StatusPass TestStatus = iota
	StatusFail
	StatusSkip
)

func (s TestStatus) String() string {
	switch s {
	case StatusPass:
		return "PASS"
	case StatusFail:
		return "FAIL"
	case StatusSkip:
		return "SKIP"
	default:
		return "UNKNOWN"
	}
}

// TestContext provides context and utilities for a single test
type TestContext struct {
	suite  *TestSuite
	name   string
	buffer *bytes.Buffer
}

// HTTPResponse represents a complete HTTP response
type HTTPResponse struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
}

// NewTestSuite creates a new test suite
func NewTestSuite(name, description, specURL string) *TestSuite {
	return &TestSuite{
		Name:        name,
		Description: description,
		SpecURL:     specURL,
		Tests:       []Test{},
		Results:     &TestResults{},
		ServerURL:   "http://localhost:9090",
		Out:         os.Stdout,
		Client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// AddTest adds a test to the suite
func (s *TestSuite) AddTest(name, description string, fn func(*TestContext) error) {
	s.Tests = append(s.Tests, Test{
		Name:        name,
		Description: description,
		Fn:          fn,
	})
}

// AddTestWithCapabilities adds a test whose optional feature dependencies are
// evaluated independently from the rest of its suite. This avoids skipping
// unrelated tests when a suite covers more than one capability.
func (s *TestSuite) AddTestWithCapabilities(name, description string, required []RequiredCapability, fn func(*TestContext) error) {
	s.Tests = append(s.Tests, Test{
		Name:                 name,
		Description:          description,
		RequiredCapabilities: required,
		Fn:                   fn,
	})
}

// Run executes all tests in the suite
func (s *TestSuite) Run() error {
	if s.Verbose {
		fmt.Fprintln(s.Out, "======================================")
		fmt.Fprintln(s.Out, "OData v4 Compliance Test")
		fmt.Fprintf(s.Out, "Suite: %s\n", s.Name)
		fmt.Fprintln(s.Out, "======================================")
		fmt.Fprintln(s.Out)
		fmt.Fprintf(s.Out, "Description: %s\n", s.Description)
		fmt.Fprintln(s.Out)
		fmt.Fprintf(s.Out, "Spec Reference: %s\n", s.SpecURL)
		fmt.Fprintln(s.Out)
	} else if !s.Quiet {
		// In non-verbose mode, show a simple progress message unless suppressed
		fmt.Fprintf(s.Out, "Running %d tests... ", len(s.Tests))
	}

	// Capability gate: if the service declares a required capability as unsupported and
	// strict mode is off, skip every test in this suite rather than running and failing them.
	if s.Capabilities != nil && !s.Strict && len(s.RequiredCapabilities) > 0 {
		for _, req := range s.RequiredCapabilities {
			if !s.Capabilities.Supports(req) {
				reason := s.Capabilities.SkipReason(req)
				for _, test := range s.Tests {
					s.Results.Total++
					s.Results.Skipped++
					s.Results.Details = append(s.Results.Details, TestDetail{
						Name:   test.Description,
						Status: StatusSkip,
						Error:  reason,
					})
				}
				if s.Verbose {
					fmt.Fprintf(s.Out, "⊘ Suite skipped: %s\n\n", reason)
				} else if !s.Quiet {
					fmt.Fprintf(s.Out, "Skipped (%s)\n", reason)
				}
				return nil
			}
		}
	}

	// If every test in a mixed-capability suite is gated, record the skips
	// without requiring the service-specific reseed action.
	if s.Capabilities != nil && !s.Strict && len(s.Tests) > 0 {
		allUnsupported := true
		reasons := make([]string, len(s.Tests))
		for i, test := range s.Tests {
			reason, unsupported := s.unsupportedCapability(test.RequiredCapabilities)
			if !unsupported {
				allUnsupported = false
				break
			}
			reasons[i] = reason
		}
		if allUnsupported {
			for i, test := range s.Tests {
				s.Results.Total++
				s.Results.Skipped++
				s.Results.Details = append(s.Results.Details, TestDetail{
					Name:   test.Description,
					Status: StatusSkip,
					Error:  reasons[i],
				})
			}
			if !s.Quiet {
				fmt.Fprintln(s.Out, "Skipped (all test capabilities are declared unsupported)")
			}
			return nil
		}
	}

	// Reseed the database once at the beginning of the suite to ensure clean state
	// Tests within a suite may depend on data created by previous tests
	if err := s.reseedDatabase(); err != nil {
		setupErr := fmt.Errorf("suite setup failed: could not reseed reference data: %w", err)
		for _, test := range s.Tests {
			s.Results.Total++
			s.Results.Failed++
			s.Results.Details = append(s.Results.Details, TestDetail{
				Name:   test.Description,
				Status: StatusFail,
				Error:  setupErr.Error(),
			})
		}
		if s.Verbose {
			fmt.Fprintf(s.Out, "\n✗ SETUP FAILURE: %v\n", setupErr)
		} else if !s.Quiet {
			fmt.Fprintln(s.Out, "Failed")
		}
		if !s.Quiet {
			s.PrintSummary()
		}
		return setupErr
	}

	for i, test := range s.Tests {
		s.Results.Total++
		if s.Capabilities != nil && !s.Strict {
			if reason, unsupported := s.unsupportedCapability(test.RequiredCapabilities); unsupported {
				s.Results.Skipped++
				s.Results.Details = append(s.Results.Details, TestDetail{
					Name:   test.Description,
					Status: StatusSkip,
					Error:  reason,
				})
				if s.Verbose {
					fmt.Fprintf(s.Out, "\n⊘ SKIP: %s\n", test.Description)
					fmt.Fprintf(s.Out, "  Reason: %s\n", reason)
				}
				continue
			}
		}
		ctx := &TestContext{
			suite:  s,
			name:   test.Name,
			buffer: &bytes.Buffer{},
		}

		err := test.Fn(ctx)
		if err != nil {
			if skipErr, ok := err.(*SkipError); ok {
				s.Results.Skipped++
				s.Results.Details = append(s.Results.Details, TestDetail{
					Name:   test.Description,
					Status: StatusSkip,
					Error:  skipErr.Reason,
				})
				if s.Verbose {
					fmt.Fprintf(s.Out, "\n⊘ SKIP: %s\n", test.Description)
					fmt.Fprintf(s.Out, "  Reason: %s\n", skipErr.Reason)
				}
			} else {
				s.Results.Failed++
				s.Results.Details = append(s.Results.Details, TestDetail{
					Name:   test.Description,
					Status: StatusFail,
					Error:  err.Error(),
				})
				if s.Verbose {
					fmt.Fprintf(s.Out, "\n✗ FAIL: %s\n", test.Description)
					fmt.Fprintf(s.Out, "  Details: %s\n", err.Error())
				}
			}
		} else {
			s.Results.Passed++
			s.Results.Details = append(s.Results.Details, TestDetail{
				Name:   test.Description,
				Status: StatusPass,
			})
			if s.Verbose {
				fmt.Fprintf(s.Out, "\n✓ PASS: %s\n", test.Description)
			}
		}

		// Print progress dots in non-verbose, non-quiet mode
		if !s.Verbose && !s.Quiet && (i+1)%10 == 0 {
			fmt.Fprintf(s.Out, "%d/%d ", i+1, len(s.Tests))
		}
	}

	if !s.Verbose && !s.Quiet {
		fmt.Fprintln(s.Out, "Done")
	}

	if !s.Quiet {
		s.PrintSummary()
	}

	if s.Results.Failed > 0 {
		return fmt.Errorf("%d test(s) failed", s.Results.Failed)
	}
	return nil
}

func (s *TestSuite) unsupportedCapability(required []RequiredCapability) (string, bool) {
	for _, req := range required {
		if !s.Capabilities.Supports(req) {
			return s.Capabilities.SkipReason(req), true
		}
	}
	return "", false
}

// PrintSummary prints the test summary in standardized format
func (s *TestSuite) PrintSummary() {
	fmt.Fprintln(s.Out)
	fmt.Fprintln(s.Out, "======================================")
	fmt.Fprintf(s.Out, "COMPLIANCE_TEST_RESULT:PASSED=%d:FAILED=%d:SKIPPED=%d:TOTAL=%d\n",
		s.Results.Passed, s.Results.Failed, s.Results.Skipped, s.Results.Total)
	fmt.Fprintln(s.Out, "======================================")

	if s.Results.Failed == 0 {
		if s.Results.AllSkipped() {
			fmt.Fprintln(s.Out, "Status: SKIPPED (no tests executed)")
		} else {
			fmt.Fprintln(s.Out, "Status: PASSING")
		}
	} else {
		fmt.Fprintln(s.Out, "Status: FAILING")

		// Print failed tests list
		if !s.Verbose && s.Results.Failed > 0 {
			fmt.Fprintln(s.Out)
			fmt.Fprintln(s.Out, "Failed Tests:")
			for _, detail := range s.Results.Details {
				if detail.Status == StatusFail {
					fmt.Fprintf(s.Out, "  ✗ %s\n", detail.Name)
					if detail.Error != "" {
						fmt.Fprintf(s.Out, "    Error: %s\n", detail.Error)
					}
				}
			}
		}
	}
}

// reseedDatabase calls the Reseed action to reset the database to a clean state
func (s *TestSuite) reseedDatabase() error {
	url := s.ServerURL + "/Reseed"
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.Client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call Reseed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Reseed returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// SkipError represents a skipped test
type SkipError struct {
	Reason string
}

func (e *SkipError) Error() string {
	return e.Reason
}

// Skip marks a test as skipped
func (c *TestContext) Skip(reason string) error {
	return &SkipError{Reason: reason}
}

// GET performs an HTTP GET request
func (c *TestContext) GET(path string, headers ...Header) (*HTTPResponse, error) {
	return c.request("GET", path, nil, headers...)
}

// POST performs an HTTP POST request
func (c *TestContext) POST(path string, body interface{}, headers ...Header) (*HTTPResponse, error) {
	return c.request("POST", path, body, headers...)
}

// PATCH performs an HTTP PATCH request
func (c *TestContext) PATCH(path string, body interface{}, headers ...Header) (*HTTPResponse, error) {
	return c.request("PATCH", path, body, headers...)
}

// PATCHRawNoContentType performs an HTTP PATCH request with raw bytes and no default Content-Type.
func (c *TestContext) PATCHRawNoContentType(path string, body []byte, headers ...Header) (*HTTPResponse, error) {
	return c.requestWithOptions("PATCH", path, body, requestOptions{skipDefaultContentType: true}, headers...)
}

// PUT performs an HTTP PUT request
func (c *TestContext) PUT(path string, body interface{}, headers ...Header) (*HTTPResponse, error) {
	return c.request("PUT", path, body, headers...)
}

// DELETE performs an HTTP DELETE request
func (c *TestContext) DELETE(path string, headers ...Header) (*HTTPResponse, error) {
	return c.request("DELETE", path, nil, headers...)
}

// HEAD performs an HTTP HEAD request
func (c *TestContext) HEAD(path string, headers ...Header) (*HTTPResponse, error) {
	return c.request("HEAD", path, nil, headers...)
}

// POSTRaw performs an HTTP POST request with raw bytes and content type
func (c *TestContext) POSTRaw(path string, body []byte, contentType string, headers ...Header) (*HTTPResponse, error) {
	hdrs := append([]Header{{Key: "Content-Type", Value: contentType}}, headers...)
	return c.request("POST", path, body, hdrs...)
}

// PUTRaw performs an HTTP PUT request with raw bytes and content type
func (c *TestContext) PUTRaw(path string, body []byte, contentType string, headers ...Header) (*HTTPResponse, error) {
	hdrs := append([]Header{{Key: "Content-Type", Value: contentType}}, headers...)
	return c.request("PUT", path, body, hdrs...)
}

// GETWithHeaders performs an HTTP GET request with custom headers
func (c *TestContext) GETWithHeaders(path string, customHeaders map[string]string) (*HTTPResponse, error) {
	headers := make([]Header, 0, len(customHeaders))
	for k, v := range customHeaders {
		headers = append(headers, Header{Key: k, Value: v})
	}
	return c.request("GET", path, nil, headers...)
}

// Log logs a message during test execution
func (c *TestContext) Log(message string) {
	if c.suite.Debug {
		fmt.Fprintf(c.suite.Out, "[LOG] %s\n", message)
	}
}

// ServerURL returns the base URL of the running compliance server
func (c *TestContext) ServerURL() string {
	return c.suite.ServerURL
}

// Header represents an HTTP header key-value pair
type Header struct {
	Key   string
	Value string
}

// request performs an HTTP request
func (c *TestContext) request(method, path string, body interface{}, headers ...Header) (*HTTPResponse, error) {
	return c.requestWithOptions(method, path, body, requestOptions{}, headers...)
}

type requestOptions struct {
	skipDefaultContentType bool
}

func (c *TestContext) requestWithOptions(method, path string, body interface{}, options requestOptions, headers ...Header) (*HTTPResponse, error) {
	// Normalize query strings so callers don't need to URL encode manually
	if strings.Contains(path, "?") {
		if parsed, err := url.Parse(path); err == nil {
			encodedPath := parsed.Path
			if rawQuery := parsed.Query().Encode(); rawQuery != "" {
				encodedPath += "?" + rawQuery
			}
			path = encodedPath
		}
	}

	url := c.suite.ServerURL + path

	var bodyReader io.Reader
	if body != nil {
		var bodyBytes []byte
		var err error
		switch v := body.(type) {
		case string:
			bodyBytes = []byte(v)
		case []byte:
			bodyBytes = v
		default:
			bodyBytes, err = json.Marshal(body)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal body: %w", err)
			}
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	contentTypeSet := false
	for _, h := range headers {
		if strings.EqualFold(h.Key, "Content-Type") {
			contentTypeSet = true
		}
		req.Header.Set(h.Key, h.Value)
	}
	if body != nil && !contentTypeSet && !options.skipDefaultContentType {
		// Default to JSON for structured payloads
		req.Header.Set("Content-Type", "application/json")
	}

	if c.suite.Debug {
		c.debugRequest(req, body)
	}

	resp, err := c.suite.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to perform request: %w", err)
	}
	//nolint:errcheck
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	httpResp := &HTTPResponse{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
		Body:       respBody,
	}

	if c.suite.Debug {
		c.debugResponse(httpResp)
	}

	return httpResp, nil
}

// debugRequest prints debug information about the request
func (c *TestContext) debugRequest(req *http.Request, body interface{}) {
	out := c.suite.Out
	fmt.Fprintln(out, "\n╔══════════════════════════════════════════════════════╗")
	fmt.Fprintln(out, "║ DEBUG: HTTP Request                                  ║")
	fmt.Fprintln(out, "╚══════════════════════════════════════════════════════╝")
	fmt.Fprintln(out)
	fmt.Fprintf(out, "Method: %s\n", req.Method)
	fmt.Fprintf(out, "URL: %s\n", req.URL.String())
	if len(req.Header) > 0 {
		fmt.Fprintln(out, "Headers:")
		for k, v := range req.Header {
			fmt.Fprintf(out, "  %s: %s\n", k, strings.Join(v, ", "))
		}
	}
	if body != nil {
		fmt.Fprintln(out, "Body:")
		if str, ok := body.(string); ok {
			fmt.Fprintln(out, str)
		} else if b, ok := body.([]byte); ok {
			fmt.Fprintln(out, string(b))
		} else {
			bodyJSON, err := json.MarshalIndent(body, "", "  ")
			if err == nil {
				fmt.Fprintln(out, string(bodyJSON))
			}
		}
	}
	fmt.Fprintln(out)
}

// debugResponse prints debug information about the response
func (c *TestContext) debugResponse(resp *HTTPResponse) {
	out := c.suite.Out
	fmt.Fprintln(out, "\n╔══════════════════════════════════════════════════════╗")
	fmt.Fprintln(out, "║ DEBUG: HTTP Response                                 ║")
	fmt.Fprintln(out, "╚══════════════════════════════════════════════════════╝")
	fmt.Fprintln(out)
	fmt.Fprintf(out, "Status Code: %d\n", resp.StatusCode)
	if len(resp.Body) > 0 {
		fmt.Fprintln(out, "Body:")
		// Try to pretty-print JSON
		var jsonBody interface{}
		if err := json.Unmarshal(resp.Body, &jsonBody); err == nil {
			prettyJSON, marshalErr := json.MarshalIndent(jsonBody, "", "  ")
			if marshalErr == nil {
				fmt.Fprintln(out, string(prettyJSON))
			}
		} else {
			fmt.Fprintln(out, string(resp.Body))
		}
	}
	fmt.Fprintln(out)
}

// AssertStatusCode checks if the response has the expected status code
func (c *TestContext) AssertStatusCode(resp *HTTPResponse, expected int) error {
	if resp.StatusCode != expected {
		// Include response body in error message for debugging
		bodyPreview := string(resp.Body)
		if len(bodyPreview) > 200 {
			bodyPreview = bodyPreview[:200] + "..."
		}
		return fmt.Errorf("status code: %d (expected %d), body: %s", resp.StatusCode, expected, bodyPreview)
	}
	return nil
}

// AssertODataError validates an OData JSON error payload for a specific HTTP status.
func (c *TestContext) AssertODataError(resp *HTTPResponse, expectedStatus int, messageFragment string) error {
	if err := c.AssertStatusCode(resp, expectedStatus); err != nil {
		return err
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(resp.Body, &payload); err != nil {
		return fmt.Errorf("expected JSON error response, got parse error: %w", err)
	}

	errObjRaw, ok := payload["error"]
	if !ok {
		return fmt.Errorf("missing error object in response")
	}

	errObj, ok := errObjRaw.(map[string]interface{})
	if !ok {
		return fmt.Errorf("error object has unexpected type %T", errObjRaw)
	}

	code, ok := errObj["code"].(string)
	if !ok || strings.TrimSpace(code) == "" {
		return fmt.Errorf("error object must include a non-empty code")
	}
	// Per OData-JSON Format §19, "code" is a service-defined, language-independent
	// string (e.g. "EntityNotFound"); it is NOT required to equal the HTTP status
	// code. We only require it to be present and non-empty.

	messages, err := collectODataErrorMessages(errObj)
	if err != nil {
		return err
	}

	if messageFragment == "" {
		return nil
	}

	needle := strings.ToLower(strings.TrimSpace(messageFragment))
	for _, message := range messages {
		if strings.Contains(strings.ToLower(message), needle) {
			return nil
		}
	}

	return fmt.Errorf("error payload does not contain expected message fragment %q; messages=%v", messageFragment, messages)
}

func collectODataErrorMessages(errObj map[string]interface{}) ([]string, error) {
	messages := []string{}

	topLevelMessage, err := extractODataMessage(errObj["message"])
	if err != nil {
		return nil, err
	}
	messages = append(messages, topLevelMessage)

	if detailsRaw, ok := errObj["details"]; ok {
		details, ok := detailsRaw.([]interface{})
		if !ok {
			return nil, fmt.Errorf("error details has unexpected type %T", detailsRaw)
		}
		for i, detailRaw := range details {
			detail, ok := detailRaw.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("error detail %d has unexpected type %T", i, detailRaw)
			}
			detailMessage, err := extractODataMessage(detail["message"])
			if err != nil {
				return nil, fmt.Errorf("invalid error detail %d message: %w", i, err)
			}
			messages = append(messages, detailMessage)
		}
	}

	return messages, nil
}

func extractODataMessage(value interface{}) (string, error) {
	switch message := value.(type) {
	case string:
		message = strings.TrimSpace(message)
		if message == "" {
			return "", fmt.Errorf("error message is empty")
		}
		return message, nil
	case map[string]interface{}:
		text, ok := message["value"].(string)
		if !ok || strings.TrimSpace(text) == "" {
			return "", fmt.Errorf("error message object must contain non-empty value")
		}
		return strings.TrimSpace(text), nil
	default:
		return "", fmt.Errorf("error message has unexpected type %T", value)
	}
}

// AssertHeader checks if the response has the expected header value
func (c *TestContext) AssertHeader(resp *HTTPResponse, key, expected string) error {
	actual := resp.Headers.Get(key)
	if actual != expected {
		return fmt.Errorf("header %s: %q (expected %q)", key, actual, expected)
	}
	return nil
}

// AssertHeaderContains checks if the response header contains the expected substring
func (c *TestContext) AssertHeaderContains(resp *HTTPResponse, key, expected string) error {
	actual := resp.Headers.Get(key)
	if !strings.Contains(actual, expected) {
		return fmt.Errorf("header %s: %q does not contain %q", key, actual, expected)
	}
	return nil
}

// AssertJSONField checks if the response body contains the expected JSON field
func (c *TestContext) AssertJSONField(resp *HTTPResponse, field string) error {
	var data map[string]interface{}
	if err := json.Unmarshal(resp.Body, &data); err != nil {
		return fmt.Errorf("invalid JSON response: %w", err)
	}

	if _, ok := data[field]; !ok {
		return fmt.Errorf("field %q not found in JSON response", field)
	}
	return nil
}

// AssertBodyContains checks if the response body contains the expected string
func (c *TestContext) AssertBodyContains(resp *HTTPResponse, expected string) error {
	if !strings.Contains(string(resp.Body), expected) {
		return fmt.Errorf("expected %q not found in response body", expected)
	}
	return nil
}

// GetJSON unmarshals the response body into the provided interface
func (c *TestContext) GetJSON(resp *HTTPResponse, v interface{}) error {
	return json.Unmarshal(resp.Body, v)
}

// IsValidJSON checks if the response body is valid JSON
func (c *TestContext) IsValidJSON(resp *HTTPResponse) bool {
	var data interface{}
	return json.Unmarshal(resp.Body, &data) == nil
}

// NewError creates a new error with the given message
func NewError(message string) error {
	return fmt.Errorf("%s", message)
}

// ContainsAny checks if a string contains any of the provided substrings
func ContainsAny(s string, substrs ...string) bool {
	for _, substr := range substrs {
		if strings.Contains(s, substr) {
			return true
		}
	}
	return false
}

// ParseEntityCollection parses a JSON response and returns the top-level OData "value" collection.
func (c *TestContext) ParseEntityCollection(resp *HTTPResponse) ([]map[string]interface{}, error) {
	var payload map[string]interface{}
	if err := json.Unmarshal(resp.Body, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	rawValue, ok := payload["value"]
	if !ok {
		return nil, fmt.Errorf("response missing 'value' collection")
	}

	rawItems, ok := rawValue.([]interface{})
	if !ok {
		return nil, fmt.Errorf("response 'value' is not an array")
	}

	items := make([]map[string]interface{}, 0, len(rawItems))
	for i, raw := range rawItems {
		entity, ok := raw.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("response item %d is not an object", i)
		}
		items = append(items, entity)
	}

	return items, nil
}

// AssertMinCollectionSize ensures a collection has at least min items.
func (c *TestContext) AssertMinCollectionSize(items []map[string]interface{}, min int) error {
	if len(items) < min {
		return fmt.Errorf("expected at least %d item(s), got %d", min, len(items))
	}
	return nil
}

// AssertEntityHasFields ensures all required fields are present on an entity.
func (c *TestContext) AssertEntityHasFields(entity map[string]interface{}, requiredFields ...string) error {
	for _, field := range requiredFields {
		if _, ok := entity[field]; !ok {
			return fmt.Errorf("required field %q is missing", field)
		}
	}
	return nil
}

// AssertEntityOnlyAllowedFields ensures entity fields are in the provided allow-list.
// OData control annotations (e.g. "@odata.id", "@odata.etag") and property
// annotations (e.g. "Photo@odata.mediaReadLink") are not structural or
// navigation properties, are not subject to $select, and are always permitted
// regardless of the allow-list; callers only need to list the structural and
// navigation property names they expect (including key properties, which
// services return even when not explicitly selected).
func (c *TestContext) AssertEntityOnlyAllowedFields(entity map[string]interface{}, allowedFields ...string) error {
	allowed := make(map[string]struct{}, len(allowedFields))
	for _, field := range allowedFields {
		allowed[field] = struct{}{}
	}

	for key := range entity {
		if strings.Contains(key, "@") {
			continue
		}
		if _, ok := allowed[key]; !ok {
			return fmt.Errorf("field %q is not allowed in this response", key)
		}
	}

	return nil
}

// AssertAllEntitiesSatisfy checks a predicate against every entity in a collection.
func (c *TestContext) AssertAllEntitiesSatisfy(
	items []map[string]interface{},
	description string,
	predicate func(entity map[string]interface{}) (bool, string),
) error {
	for i, entity := range items {
		ok, reason := predicate(entity)
		if !ok {
			if reason == "" {
				reason = "predicate returned false"
			}
			return fmt.Errorf("entity at index %d does not satisfy %s: %s", i, description, reason)
		}
	}
	return nil
}

// AssertEntitiesSortedByFloat checks ascending/descending ordering by a float field.
func (c *TestContext) AssertEntitiesSortedByFloat(items []map[string]interface{}, field string, ascending bool) error {
	if len(items) < 2 {
		return nil
	}

	for i := 1; i < len(items); i++ {
		prev, ok := items[i-1][field].(float64)
		if !ok {
			return fmt.Errorf("item %d field %q is missing or not numeric", i-1, field)
		}
		curr, ok := items[i][field].(float64)
		if !ok {
			return fmt.Errorf("item %d field %q is missing or not numeric", i, field)
		}

		if ascending && curr < prev {
			return fmt.Errorf("items not sorted ascending by %q at index %d: %.6f < %.6f", field, i, curr, prev)
		}
		if !ascending && curr > prev {
			return fmt.Errorf("items not sorted descending by %q at index %d: %.6f > %.6f", field, i, curr, prev)
		}
	}

	return nil
}
