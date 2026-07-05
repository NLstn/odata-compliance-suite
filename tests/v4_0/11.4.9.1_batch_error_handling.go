package v4_0

import (
	"fmt"
	"strings"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// BatchErrorHandling creates the 11.4.9.1 Batch Error Handling test suite
func BatchErrorHandling() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.4.9.1 Batch Error Handling",
		"Tests error handling in batch requests including changeset atomicity, error responses, and malformed requests.",
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

	// Test 1: Batch endpoint responds to malformed boundary
	suite.AddTest(
		"test_malformed_boundary",
		"Malformed batch boundary handled correctly",
		func(ctx *framework.TestContext) error {
			segment, err := getProductSegment(ctx, 0)
			if err != nil {
				return err
			}
			batchBody := fmt.Sprintf(`--wrong_boundary
Content-Type: application/http
Content-Transfer-Encoding: binary

GET %s HTTP/1.1
Accept: application/json


--wrong_boundary--`, segment)

			resp, err := ctx.POSTRaw("/$batch", []byte(batchBody), "multipart/mixed; boundary=batch_boundary")
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, 400)
		},
	)

	// Test 2: Independent requests don't affect each other
	suite.AddTest(
		"test_independent_requests",
		"Independent requests don't affect each other",
		func(ctx *framework.TestContext) error {
			firstSegment, err := getProductSegment(ctx, 0)
			if err != nil {
				return err
			}
			secondSegment, err := getProductSegment(ctx, 1)
			if err != nil {
				return err
			}
			invalidSegment := nonExistingEntitySegment("Products")
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


--batch_boundary
Content-Type: application/http
Content-Transfer-Encoding: binary

GET %s HTTP/1.1
Accept: application/json


--batch_boundary--`, firstSegment, invalidSegment, secondSegment)

			resp, err := ctx.POSTRaw("/$batch", []byte(batchBody), "multipart/mixed; boundary=batch_boundary")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			// Should have both 200 and 404 responses
			successCount := strings.Count(string(resp.Body), "HTTP/1.1 200")
			notFoundCount := strings.Count(string(resp.Body), "HTTP/1.1 404")

			if successCount < 2 || notFoundCount < 1 {
				return framework.NewError("Expected at least two 200 responses and one 404 response")
			}

			return nil
		},
	)

	// Test 3: Error responses in batch have proper format
	suite.AddTest(
		"test_error_format_in_batch",
		"Error responses in batch have proper format",
		func(ctx *framework.TestContext) error {
			invalidSegment := nonExistingEntitySegment("Products")
			batchBody := fmt.Sprintf(`--batch_boundary
Content-Type: application/http
Content-Transfer-Encoding: binary

GET %s HTTP/1.1
Accept: application/json


--batch_boundary--`, invalidSegment)

			resp, err := ctx.POSTRaw("/$batch", []byte(batchBody), "multipart/mixed; boundary=batch_boundary")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			body := string(resp.Body)
			// Batch response MUST embed a 404 sub-response for the non-existent entity.
			if !strings.Contains(body, "HTTP/1.1 404") {
				return framework.NewError("batch sub-response for non-existent entity must include an HTTP/1.1 404 status line")
			}
			// Sub-response body MUST be a structured OData error with "error", "code", and "message"
			// per OData JSON Format §9.
			if !strings.Contains(body, `"error"`) {
				return framework.NewError(`batch error sub-response must contain an "error" key per OData JSON Format §9`)
			}
			if !strings.Contains(body, `"code"`) || !strings.Contains(body, `"message"`) {
				return framework.NewError(`batch error object must contain "code" and "message" properties per OData JSON Format §9.3`)
			}
			return nil
		},
	)

	// Test 4: Changeset atomicity — failed changeset must roll back all prior
	// operations within the same changeset (OData §11.4.9 requires atomic execution).
	suite.AddTest(
		"test_changeset_atomicity_rollback",
		"Failed changeset rolls back all prior operations within the changeset (§11.4.9)",
		func(ctx *framework.TestContext) error {
			ids, err := fetchEntityIDs(ctx, "Products", 1)
			if err != nil {
				return err
			}
			if len(ids) == 0 {
				return fmt.Errorf("need at least one product for changeset atomicity test")
			}
			productID := ids[0]
			productPath := fmt.Sprintf("/Products(%s)", productID)

			// Record original Name so we can verify rollback.
			getResp, err := ctx.GET(productPath)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(getResp, 200); err != nil {
				return err
			}
			var originalEntity map[string]interface{}
			if err := ctx.GetJSON(getResp, &originalEntity); err != nil {
				return err
			}
			originalName, _ := originalEntity["Name"].(string)

			// Changeset: valid PATCH on real product followed by DELETE on nonexistent
			// entity (must 404). If the server supports changeset atomicity the PATCH
			// must be rolled back.
			changedName := "Atomicity Test Should Be Rolled Back"
			batchBoundary := "batch_atomicity"
			changesetBoundary := "changeset_atomicity"
			batchBody := fmt.Sprintf(`--%[1]s
Content-Type: multipart/mixed; boundary=%[2]s

--%[2]s
Content-Type: application/http
Content-Transfer-Encoding: binary

PATCH %[3]s HTTP/1.1
Content-Type: application/json

{"Name":%[4]q}

--%[2]s
Content-Type: application/http
Content-Transfer-Encoding: binary

DELETE /Products(%[5]s) HTTP/1.1


--%[2]s--

--%[1]s--`,
				batchBoundary, changesetBoundary,
				productPath, changedName,
				nonExistingUUID)

			resp, err := ctx.POSTRaw("/$batch", []byte(batchBody),
				fmt.Sprintf("multipart/mixed; boundary=%s", batchBoundary))
			if err != nil {
				return err
			}
			// The outer batch envelope always returns 200 even when a changeset fails.
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			// The DELETE on the nonexistent entity must produce a 4xx sub-response.
			body := string(resp.Body)
			if !strings.Contains(body, "HTTP/1.1 4") {
				return framework.NewError(
					"expected a 4xx sub-response for DELETE on nonexistent entity; " +
						"changeset did not fail as required")
			}

			// Verify atomicity: the PATCH on the real product must have been rolled back.
			verifyResp, err := ctx.GET(productPath)
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(verifyResp, 200); err != nil {
				return err
			}
			var currentEntity map[string]interface{}
			if err := ctx.GetJSON(verifyResp, &currentEntity); err != nil {
				return err
			}
			currentName, _ := currentEntity["Name"].(string)
			if currentName == changedName {
				return framework.NewError(
					fmt.Sprintf("changeset atomicity violated: PATCH was not rolled back after "+
						"changeset failure; product Name is %q but should still be %q "+
						"(OData §11.4.9 requires atomic changesets)",
						currentName, originalName))
			}

			return nil
		},
	)

	return suite
}
