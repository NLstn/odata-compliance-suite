package framework_test

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/nlstn/odata-compliance-suite/framework"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestSuiteRunFailsClosedWhenReseedFails(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost || req.URL.Path != "/Reseed" {
			t.Errorf("unexpected request %s %s", req.Method, req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("reseed unavailable")),
			Request:    req,
		}, nil
	})}

	suite := framework.NewTestSuite("setup", "setup", "https://example.test/spec")
	suite.ServerURL = "http://example.test"
	suite.Client = client
	suite.Out = io.Discard
	suite.Quiet = true
	testRan := false
	suite.AddTest("would_run", "would run", func(*framework.TestContext) error {
		testRan = true
		return nil
	})

	err := suite.Run()
	if err == nil || !strings.Contains(err.Error(), "could not reseed reference data") {
		t.Fatalf("Run() error = %v, want reseed setup failure", err)
	}
	if testRan {
		t.Fatal("test body ran against stale data after reseed failed")
	}
	if suite.Results.Total != 1 || suite.Results.Failed != 1 || suite.Results.Passed != 0 {
		t.Fatalf("unexpected results: %+v", suite.Results)
	}
	if len(suite.Results.Details) != 1 || !strings.Contains(suite.Results.Details[0].Error, "suite setup failed") {
		t.Fatalf("unexpected failure details: %+v", suite.Results.Details)
	}
}

func TestSuiteRunGatesCapabilitiesPerTest(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost || req.URL.Path != "/Reseed" {
			t.Errorf("unexpected request %s %s", req.Method, req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{}`)),
			Request:    req,
		}, nil
	})}

	profile := framework.NewCapabilityProfile()
	profile.SetEntitySetCap("Products", framework.CapSort, false)

	suite := framework.NewTestSuite("mixed", "mixed", "https://example.test/spec")
	suite.ServerURL = "http://example.test"
	suite.Client = client
	suite.Out = io.Discard
	suite.Quiet = true
	suite.Capabilities = profile

	selectRan := false
	sortRan := false
	suite.AddTestWithCapabilities("select", "select", []framework.RequiredCapability{
		framework.Require(framework.CapSelect, "Products"),
	}, func(*framework.TestContext) error {
		selectRan = true
		return nil
	})
	suite.AddTestWithCapabilities("sort", "sort", []framework.RequiredCapability{
		framework.Require(framework.CapSort, "Products"),
	}, func(*framework.TestContext) error {
		sortRan = true
		return nil
	})

	if err := suite.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !selectRan || sortRan {
		t.Fatalf("selectRan=%v sortRan=%v, want true/false", selectRan, sortRan)
	}
	if suite.Results.Passed != 1 || suite.Results.Skipped != 1 || suite.Results.Failed != 0 {
		t.Fatalf("unexpected results: %+v", suite.Results)
	}
}

func TestSuiteRunDoesNotReseedWhenAllPerTestCapabilitiesAreUnsupported(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		t.Fatalf("unexpected request %s %s", req.Method, req.URL.Path)
		return nil, nil
	})}

	profile := framework.NewCapabilityProfile()
	profile.SetEntitySetCap("Products", framework.CapSelect, false)

	suite := framework.NewTestSuite("gated", "gated", "https://example.test/spec")
	suite.ServerURL = "http://example.test"
	suite.Client = client
	suite.Out = io.Discard
	suite.Quiet = true
	suite.Capabilities = profile
	suite.AddTestWithCapabilities("select", "select", []framework.RequiredCapability{
		framework.Require(framework.CapSelect, "Products"),
	}, func(*framework.TestContext) error {
		t.Fatal("gated test ran")
		return nil
	})

	if err := suite.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if suite.Results.Skipped != 1 || suite.Results.Total != 1 {
		t.Fatalf("unexpected results: %+v", suite.Results)
	}
}
