package framework

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

type nextLinkTransport func(*http.Request) (*http.Response, error)

func (f nextLinkTransport) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestGETNextLinkPreservesOpaqueURLAndServicePath(t *testing.T) {
	var got []string
	client := &http.Client{Transport: nextLinkTransport(func(req *http.Request) (*http.Response, error) {
		got = append(got, req.URL.String())
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"value":[]}`)), Request: req}, nil
	})}
	ctx := &TestContext{suite: &TestSuite{ServerURL: "http://example.test/odata", Client: client}}
	for _, link := range []string{
		"http://example.test/odata/Products?%24skiptoken=a%2Bb%2Fc&%24filter=Name%20eq%20%27A%27",
		"Products?%24skiptoken=a%2Bb%2Fc&%24filter=Name%20eq%20%27A%27",
	} {
		if _, err := ctx.GETNextLink(link); err != nil {
			t.Fatal(err)
		}
	}
	want := "http://example.test/odata/Products?%24skiptoken=a%2Bb%2Fc&%24filter=Name%20eq%20%27A%27"
	for _, actual := range got {
		if actual != want {
			t.Errorf("nextLink request = %q, want %q", actual, want)
		}
	}
}
