package v4_0

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// BatchRequests creates the 11.4.9 Batch Requests test suite
func BatchRequests() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.4.9 Batch Requests",
		"Tests batch request processing according to OData v4 specification.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_BatchRequests",
	)

	getProductSegment := func(ctx *framework.TestContext, index int) (string, error) {
		ids, err := fetchEntityIDs(ctx, "Products", index+1)
		if err != nil {
			return "", err
		}
		if len(ids) <= index {
			return "", fmt.Errorf("need at least %d product(s)", index+1)
		}
		return fmt.Sprintf("Products(%s)", ids[index]), nil
	}
	productJSON := func(ctx *framework.TestContext, name string) (string, error) {
		payload, err := buildProductPayload(ctx, name, 12.34)
		if err != nil {
			return "", err
		}
		body, err := json.Marshal(payload)
		if err != nil {
			return "", err
		}
		return string(body), nil
	}

	// Test 1: $batch endpoint exists
	suite.AddTest(
		"test_batch_endpoint_responds",
		"$batch endpoint responds",
		func(ctx *framework.TestContext) error {
			segment, err := getProductSegment(ctx, 0)
			if err != nil {
				return err
			}
			batchBody := fmt.Sprintf(`--batch_boundary
Content-Type: application/http
Content-Transfer-Encoding: binary

GET %s HTTP/1.1
Accept: application/json


--batch_boundary--`, segment)

			resp, err := ctx.POSTRaw("/$batch", []byte(batchBody), "multipart/mixed; boundary=batch_boundary")
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, 200)
		},
	)

	// Test 2: Batch response has multipart/mixed Content-Type
	suite.AddTest(
		"test_batch_response_content_type",
		"Batch response has multipart/mixed Content-Type",
		func(ctx *framework.TestContext) error {
			segment, err := getProductSegment(ctx, 0)
			if err != nil {
				return err
			}
			batchBody := fmt.Sprintf(`--batch_boundary
Content-Type: application/http
Content-Transfer-Encoding: binary

GET %s HTTP/1.1
Accept: application/json


--batch_boundary--`, segment)

			resp, err := ctx.POSTRaw("/$batch", []byte(batchBody), "multipart/mixed; boundary=batch_boundary")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			contentType := resp.Headers.Get("Content-Type")
			if !strings.Contains(contentType, "multipart/mixed") {
				return framework.NewError("Expected multipart/mixed Content-Type")
			}

			return nil
		},
	)

	// Test 2b: Batch response includes the OData-Version header.
	// Per OData v4.0 Part 1 §8.1.5 a service MUST return OData-Version on every
	// response, including the outer $batch response.
	suite.AddTest(
		"test_batch_response_odata_version",
		"Batch response includes OData-Version header",
		func(ctx *framework.TestContext) error {
			segment, err := getProductSegment(ctx, 0)
			if err != nil {
				return err
			}
			batchBody := fmt.Sprintf(`--batch_boundary
Content-Type: application/http
Content-Transfer-Encoding: binary

GET %s HTTP/1.1
Accept: application/json


--batch_boundary--`, segment)

			resp, err := ctx.POSTRaw("/$batch", []byte(batchBody), "multipart/mixed; boundary=batch_boundary")
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			if version := strings.TrimSpace(resp.Headers.Get("OData-Version")); version == "" {
				return framework.NewError("batch response is missing the required OData-Version header")
			}

			return nil
		},
	)

	// Test 3: Batch with multiple GET requests
	suite.AddTest(
		"test_batch_multiple_gets",
		"Batch with multiple GET requests",
		func(ctx *framework.TestContext) error {
			firstSegment, err := getProductSegment(ctx, 0)
			if err != nil {
				return err
			}
			secondSegment, err := getProductSegment(ctx, 1)
			if err != nil {
				return err
			}
			batchBody := fmt.Sprintf(`--batch_boundary
Content-Type: application/http
Content-Transfer-Encoding: binary

GET %s HTTP/1.1
Accept: application/json


--batch_boundary
Content-Type: application/http
Content-Transfer-Encoding: binary

GET %s HTTP/1.1
Accept: application/json


--batch_boundary--`, firstSegment, secondSegment)

			resp, err := ctx.POSTRaw("/$batch", []byte(batchBody), "multipart/mixed; boundary=batch_boundary")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			// Check for multiple HTTP responses in body
			responseCount := strings.Count(string(resp.Body), "HTTP/1.1")
			if responseCount < 2 {
				return framework.NewError("Expected at least 2 responses in batch")
			}

			return nil
		},
	)

	// Test 4: Invalid batch request returns 400
	suite.AddTest(
		"test_batch_invalid",
		"Invalid batch request returns 400",
		func(ctx *framework.TestContext) error {
			batchBody := `--batch_boundary
INVALID CONTENT
--batch_boundary--`

			resp, err := ctx.POSTRaw("/$batch", []byte(batchBody), "multipart/mixed; boundary=batch_boundary")
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, 400)
		},
	)

	suite.AddTest(
		"test_batch_part_inherits_versions",
		"A multipart part inherits outer request and maximum versions",
		func(ctx *framework.TestContext) error {
			payload, err := productJSON(ctx, "Batch Inherited Payload Version")
			if err != nil {
				return err
			}
			batchBody := fmt.Sprintf(`--batch_version_inherit
Content-Type: application/http
Content-Transfer-Encoding: binary

POST Products HTTP/1.1
Content-Type: application/json

%s
--batch_version_inherit--`, payload)
			resp, err := ctx.POSTRaw("/$batch", []byte(batchBody),
				"multipart/mixed; boundary=batch_version_inherit",
				framework.Header{Key: "OData-Version", Value: "4.01"},
				framework.Header{Key: "OData-MaxVersion", Value: "4.0"})
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}
			body := string(resp.Body)
			if !strings.Contains(body, "HTTP/1.1 2") {
				return fmt.Errorf("inherited-version part did not succeed: %s", body)
			}
			if !strings.Contains(strings.ToLower(body), "odata-version: 4.0") {
				return fmt.Errorf("inherited OData-MaxVersion did not produce a 4.0 part response: %s", body)
			}
			return nil
		},
	)

	suite.AddTest(
		"test_batch_part_versions_override_outer",
		"Explicit multipart part versions override inherited outer values",
		func(ctx *framework.TestContext) error {
			payload, err := productJSON(ctx, "Batch Overridden Payload Version")
			if err != nil {
				return err
			}
			batchBody := fmt.Sprintf(`--batch_version_override
Content-Type: application/http
Content-Transfer-Encoding: binary

POST Products HTTP/1.1
Content-Type: application/json
OData-Version: 4.0
OData-MaxVersion: 4.0

%s
--batch_version_override--`, payload)
			resp, err := ctx.POSTRaw("/$batch", []byte(batchBody),
				"multipart/mixed; boundary=batch_version_override",
				framework.Header{Key: "OData-Version", Value: "4.01"},
				framework.Header{Key: "OData-MaxVersion", Value: "4.01"})
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}
			body := string(resp.Body)
			if !strings.Contains(body, "HTTP/1.1 2") {
				return fmt.Errorf("part with explicit versions did not succeed: %s", body)
			}
			if !strings.Contains(strings.ToLower(body), "odata-version: 4.0") {
				return fmt.Errorf("part OData-MaxVersion override did not produce a 4.0 response: %s", body)
			}
			return nil
		},
	)

	suite.AddTest(
		"test_batch_invalid_part_version",
		"An invalid multipart part payload version returns inner 4xx without mutation",
		func(ctx *framework.TestContext) error {
			const name = "Batch Invalid Payload Version"
			payload, err := productJSON(ctx, name)
			if err != nil {
				return err
			}
			batchBody := fmt.Sprintf(`--batch_invalid_version
Content-Type: application/http
Content-Transfer-Encoding: binary

POST Products HTTP/1.1
Content-Type: application/json
OData-Version: 4.02

%s
--batch_invalid_version--`, payload)
			resp, err := ctx.POSTRaw("/$batch", []byte(batchBody),
				"multipart/mixed; boundary=batch_invalid_version",
				framework.Header{Key: "OData-Version", Value: "4.0"})
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}
			if !strings.Contains(string(resp.Body), "HTTP/1.1 4") {
				return fmt.Errorf("invalid part payload version did not return inner 4xx: %s", string(resp.Body))
			}
			return assertNoProductNamed(ctx, name)
		},
	)

	suite.AddTest(
		"test_batch_invalid_changeset_version_rolls_back",
		"An invalid payload version in a changeset rolls back all mutations",
		func(ctx *framework.TestContext) error {
			const firstName = "Batch Version Rollback First"
			const secondName = "Batch Version Rollback Second"
			firstPayload, err := productJSON(ctx, firstName)
			if err != nil {
				return err
			}
			secondPayload, err := productJSON(ctx, secondName)
			if err != nil {
				return err
			}
			batchBody := fmt.Sprintf(`--batch_version_rollback
Content-Type: multipart/mixed; boundary=changeset_version_rollback

--changeset_version_rollback
Content-Type: application/http
Content-Transfer-Encoding: binary

POST Products HTTP/1.1
Content-Type: application/json

%s
--changeset_version_rollback
Content-Type: application/http
Content-Transfer-Encoding: binary

POST Products HTTP/1.1
Content-Type: application/json
OData-Version: 4.02

%s
--changeset_version_rollback--
--batch_version_rollback--`, firstPayload, secondPayload)
			resp, err := ctx.POSTRaw("/$batch", []byte(batchBody),
				"multipart/mixed; boundary=batch_version_rollback",
				framework.Header{Key: "OData-Version", Value: "4.0"})
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}
			if !strings.Contains(string(resp.Body), "HTTP/1.1 4") {
				return fmt.Errorf("changeset invalid payload version did not return inner 4xx: %s", string(resp.Body))
			}
			if err := assertNoProductNamed(ctx, firstName); err != nil {
				return fmt.Errorf("changeset did not roll back its valid request: %w", err)
			}
			return assertNoProductNamed(ctx, secondName)
		},
	)

	return suite
}
