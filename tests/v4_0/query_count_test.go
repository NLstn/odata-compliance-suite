package v4_0

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestQueryCountChecksAllPages(t *testing.T) {
	for _, tc := range []struct {
		name     string
		count    string
		wantPass bool
	}{
		{"total", "2", true},
		{"first page size", "1", false},
		{"inflated total", "3", false},
		{"fractional count", "2.5", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc("POST /Reseed", func(w http.ResponseWriter, r *http.Request) {})
			mux.HandleFunc("GET /Products", func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get("$count") != "true" {
					http.Error(w, "expected $count=true", http.StatusBadRequest)
					return
				}
				fmt.Fprintf(w, `{"@odata.count":%s,"value":[{"ID":"one"}],"@odata.nextLink":"/next"}`, tc.count)
			})
			mux.HandleFunc("GET /next", func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, `{"value":[{"ID":"two"}]}`)
			})
			server := httptest.NewServer(mux)
			defer server.Close()

			suite := QueryCount()
			suite.ServerURL = server.URL
			suite.Out = io.Discard
			for _, test := range suite.Tests {
				if test.Name == "test_count_true" {
					suite.Tests = nil
					suite.AddTest(test.Name, test.Description, test.Fn)
					err := suite.Run()
					if (err == nil) != tc.wantPass {
						t.Fatalf("Run() = %v, details = %+v", err, suite.Results.Details)
					}
					return
				}
			}
			t.Fatal("test_count_true is missing")
		})
	}
}
