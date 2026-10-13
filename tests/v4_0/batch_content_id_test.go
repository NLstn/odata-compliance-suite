package v4_0

import (
	"net/http"
	"strings"
	"testing"

	"github.com/nlstn/odata-compliance-suite/framework"
)

func TestFailedContentIDChangesetResponse(t *testing.T) {
	errorPart := "Content-Type: application/http\r\n\r\nHTTP/1.1 400 Bad Request\r\nContent-Type: application/json\r\n\r\n" + `{"error":{"code":"InvalidReference","message":"Unknown Content-ID"}}` + "\r\n"
	successPart := "Content-Type: application/http\r\n\r\nHTTP/1.1 201 Created\r\n\r\n{}\r\n"
	for _, tc := range []struct {
		name, body string
		valid      bool
	}{
		{"single error", "--batch\r\n" + errorPart + "--batch--\r\n", true},
		{"rolled back success included", "--batch\r\n" + successPart + "--batch\r\n" + errorPart + "--batch--\r\n", false},
		{"extra error", "--batch\r\n" + errorPart + "--batch\r\n" + errorPart + "--batch--\r\n", false},
		{"nested changeset", "--batch\r\nContent-Type: multipart/mixed; boundary=change\r\n\r\n--change\r\n" + errorPart + "--change--\r\n--batch--\r\n", false},
		{"missing OData error", "--batch\r\n" + strings.Replace(errorPart, `{"error":{"code":"InvalidReference","message":"Unknown Content-ID"}}`, `{}`, 1) + "--batch--\r\n", false},
		{"empty batch", "--batch--\r\n", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resp := &framework.HTTPResponse{StatusCode: 200, Headers: http.Header{"Content-Type": {"multipart/mixed; boundary=batch"}}, Body: []byte(tc.body)}
			err := assertFailedContentIDChangeset(&framework.TestContext{}, resp)
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v, error=%v", tc.valid, err)
			}
		})
	}
}
