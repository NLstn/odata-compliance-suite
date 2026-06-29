package v4_0

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// BatchChangesetAtomicity creates the 11.4.9.2 Batch Changeset Atomicity test suite.
// §11.4.9.2: A changeset is an atomic unit of work — all requests inside it MUST
// either all succeed (committed) or all fail (rolled back). Only modification
// requests (POST, PUT, PATCH, DELETE) and action invocations are allowed inside a
// changeset; GET requests are explicitly forbidden.
func BatchChangesetAtomicity() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.4.9.2 Batch Changeset Atomicity",
		"Tests atomic changeset behaviour in $batch requests: successful changesets commit all operations, failed changesets roll back all operations, and GET requests inside changesets are rejected.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_BatchRequests",
	)

	// buildProductJSON returns a minimal valid product JSON body.
	buildProductJSON := func(name string) string {
		return fmt.Sprintf(`{"Name":%q,"Price":9.99,"Status":1}`, name)
	}

	// countProductsByName fetches Products filtered by Name and returns the number of matching items.
	countProductsByName := func(ctx *framework.TestContext, name string) (int, error) {
		filter := fmt.Sprintf("Name eq '%s'", name)
		resp, err := ctx.GET(fmt.Sprintf("/Products?$filter=%s", filter))
		if err != nil {
			return 0, err
		}
		if err := ctx.AssertStatusCode(resp, 200); err != nil {
			return 0, err
		}
		var body struct {
			Value []json.RawMessage `json:"value"`
		}
		if err := json.Unmarshal(resp.Body, &body); err != nil {
			return 0, fmt.Errorf("parse Products response: %w", err)
		}
		return len(body.Value), nil
	}

	// Test 1: A changeset with all valid operations commits all entities atomically.
	suite.AddTest(
		"test_successful_changeset_commits_atomically",
		"Successful changeset commits all operations atomically",
		func(ctx *framework.TestContext) error {
			name1 := "AtomicCommitProduct1_11492"
			name2 := "AtomicCommitProduct2_11492"

			batchBody := fmt.Sprintf("--batch_outer\r\nContent-Type: multipart/mixed; boundary=changeset_1\r\n\r\n"+
				"--changeset_1\r\n"+
				"Content-Type: application/http\r\n"+
				"Content-Transfer-Encoding: binary\r\n\r\n"+
				"POST Products HTTP/1.1\r\n"+
				"Content-Type: application/json\r\n\r\n"+
				"%s\r\n"+
				"--changeset_1\r\n"+
				"Content-Type: application/http\r\n"+
				"Content-Transfer-Encoding: binary\r\n\r\n"+
				"POST Products HTTP/1.1\r\n"+
				"Content-Type: application/json\r\n\r\n"+
				"%s\r\n"+
				"--changeset_1--\r\n"+
				"--batch_outer--",
				buildProductJSON(name1), buildProductJSON(name2))

			resp, err := ctx.POSTRaw("/$batch", []byte(batchBody), "multipart/mixed; boundary=batch_outer")
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			// Both entities must be visible after the successful changeset.
			count1, err := countProductsByName(ctx, name1)
			if err != nil {
				return err
			}
			count2, err := countProductsByName(ctx, name2)
			if err != nil {
				return err
			}
			if count1 == 0 || count2 == 0 {
				return framework.NewError(fmt.Sprintf(
					"expected both products to be created, got count1=%d count2=%d", count1, count2))
			}
			return nil
		},
	)

	// Test 2: A changeset where the second operation fails must roll back the first.
	suite.AddTest(
		"test_failed_changeset_rollback",
		"Failed changeset rolls back all previously applied operations",
		func(ctx *framework.TestContext) error {
			name1 := "RollbackProduct_11492"

			// First part: valid POST.  Second part: intentionally invalid (empty body).
			batchBody := fmt.Sprintf("--batch_outer\r\nContent-Type: multipart/mixed; boundary=changeset_1\r\n\r\n"+
				"--changeset_1\r\n"+
				"Content-Type: application/http\r\n"+
				"Content-Transfer-Encoding: binary\r\n\r\n"+
				"POST Products HTTP/1.1\r\n"+
				"Content-Type: application/json\r\n\r\n"+
				"%s\r\n"+
				"--changeset_1\r\n"+
				"Content-Type: application/http\r\n"+
				"Content-Transfer-Encoding: binary\r\n\r\n"+
				"POST Products HTTP/1.1\r\n"+
				"Content-Type: application/json\r\n\r\n"+
				"{}\r\n"+ // missing required fields — must fail
				"--changeset_1--\r\n"+
				"--batch_outer--",
				buildProductJSON(name1))

			resp, err := ctx.POSTRaw("/$batch", []byte(batchBody), "multipart/mixed; boundary=batch_outer")
			if err != nil {
				return err
			}
			// Outer $batch call always returns 200 even when an inner changeset fails.
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			// The first product must NOT be present — the changeset must have been rolled back.
			count, err := countProductsByName(ctx, name1)
			if err != nil {
				return err
			}
			if count != 0 {
				return framework.NewError(fmt.Sprintf(
					"expected rollback: product %q should not exist, but found %d instance(s)", name1, count))
			}
			return nil
		},
	)

	// Test 3: A successful changeset returns a 2xx response for each operation inside,
	// wrapped in a multipart/mixed changeset response.
	suite.AddTest(
		"test_changeset_response_structure",
		"Successful changeset returns 2xx per operation with correct multipart structure",
		func(ctx *framework.TestContext) error {
			name1 := "StructureProduct1_11492"
			name2 := "StructureProduct2_11492"

			batchBody := fmt.Sprintf("--batch_outer\r\nContent-Type: multipart/mixed; boundary=changeset_1\r\n\r\n"+
				"--changeset_1\r\n"+
				"Content-Type: application/http\r\n"+
				"Content-Transfer-Encoding: binary\r\n\r\n"+
				"POST Products HTTP/1.1\r\n"+
				"Content-Type: application/json\r\n\r\n"+
				"%s\r\n"+
				"--changeset_1\r\n"+
				"Content-Type: application/http\r\n"+
				"Content-Transfer-Encoding: binary\r\n\r\n"+
				"POST Products HTTP/1.1\r\n"+
				"Content-Type: application/json\r\n\r\n"+
				"%s\r\n"+
				"--changeset_1--\r\n"+
				"--batch_outer--",
				buildProductJSON(name1), buildProductJSON(name2))

			resp, err := ctx.POSTRaw("/$batch", []byte(batchBody), "multipart/mixed; boundary=batch_outer")
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			body := string(resp.Body)

			// Outer response must be multipart.
			contentType := resp.Headers.Get("Content-Type")
			if !strings.Contains(contentType, "multipart/mixed") {
				return framework.NewError("outer batch response must have Content-Type: multipart/mixed")
			}

			// Each inner POST must have produced a 201 Created response.
			successCount := strings.Count(body, "HTTP/1.1 201")
			if successCount < 2 {
				return framework.NewError(fmt.Sprintf(
					"expected at least 2 × 'HTTP/1.1 201' inside the changeset response, got %d", successCount))
			}

			return nil
		},
	)

	// Test 4: Requests outside a changeset are independent of a failing changeset.
	suite.AddTest(
		"test_changeset_outside_requests_unaffected",
		"Requests outside a failing changeset are executed independently",
		func(ctx *framework.TestContext) error {
			// Fetch an existing product segment for the outer GET.
			ids, err := fetchEntityIDs(ctx, "Products", 1)
			if err != nil {
				return err
			}
			if len(ids) == 0 {
				return fmt.Errorf("need at least one existing product")
			}
			productSegment := fmt.Sprintf("Products(%s)", ids[0])

			// Batch: outer GET (must succeed) + failing changeset (empty body POST).
			// Use \n line endings (as in all other batch tests) so the MIME parser
			// correctly identifies boundaries even for GET parts with no body.
			batchBody := fmt.Sprintf("--batch_outer\n"+
				"Content-Type: application/http\n"+
				"Content-Transfer-Encoding: binary\n"+
				"\n"+
				"GET %s HTTP/1.1\n"+
				"Accept: application/json\n"+
				"\n"+
				"\n"+
				"--batch_outer\n"+
				"Content-Type: multipart/mixed; boundary=changeset_1\n"+
				"\n"+
				"--changeset_1\n"+
				"Content-Type: application/http\n"+
				"Content-Transfer-Encoding: binary\n"+
				"\n"+
				"POST Products HTTP/1.1\n"+
				"Content-Type: application/json\n"+
				"\n"+
				"{}\n"+ // invalid — must fail
				"\n"+
				"--changeset_1--\n"+
				"--batch_outer--",
				productSegment)

			resp, err := ctx.POSTRaw("/$batch", []byte(batchBody), "multipart/mixed; boundary=batch_outer")
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			body := string(resp.Body)

			// The outer GET must have returned 200.
			if !strings.Contains(body, "HTTP/1.1 200") {
				return framework.NewError("expected the outer GET request to succeed with 200")
			}

			// The changeset POST must have returned a 4xx or an error payload.
			innerHas4xx := strings.Contains(body, "HTTP/1.1 4")
			if !innerHas4xx && !strings.Contains(body, `"error"`) {
				return framework.NewError("expected the failing changeset to produce a 4xx or error response")
			}

			return nil
		},
	)

	// Test 5: GET inside a changeset MUST be rejected with 4xx.
	// §11.4.9.2: "All requests within a changeset MUST be either modification
	// requests or action invocation requests."
	//
	// NOTE: The go-odata reference server currently does NOT reject GET inside a
	// changeset — it returns 200 for the inner GET (confirmed spec violation).
	// A GitHub issue has been filed at NLstn/go-odata for this violation.
	// This test is intentionally written to FAIL against that server to surface
	// the non-compliance.
	suite.AddTest(
		"test_get_in_changeset_rejected",
		"GET request inside a changeset is rejected with 4xx (§11.4.9.2 forbids non-modification requests)",
		func(ctx *framework.TestContext) error {
			ids, err := fetchEntityIDs(ctx, "Products", 1)
			if err != nil {
				return err
			}
			if len(ids) == 0 {
				return fmt.Errorf("need at least one existing product")
			}
			productSegment := fmt.Sprintf("Products(%s)", ids[0])

			batchBody := fmt.Sprintf("--batch_outer\n"+
				"Content-Type: multipart/mixed; boundary=changeset_1\n"+
				"\n"+
				"--changeset_1\n"+
				"Content-Type: application/http\n"+
				"Content-Transfer-Encoding: binary\n"+
				"\n"+
				"GET %s HTTP/1.1\n"+
				"Accept: application/json\n"+
				"\n"+
				"\n"+
				"--changeset_1--\n"+
				"--batch_outer--",
				productSegment)

			resp, err := ctx.POSTRaw("/$batch", []byte(batchBody), "multipart/mixed; boundary=batch_outer")
			if err != nil {
				return err
			}

			// The server MUST reject the $batch request or the inner changeset part
			// with a 4xx status because GET is not permitted inside a changeset.
			body := string(resp.Body)
			outerIs4xx := resp.StatusCode >= 400 && resp.StatusCode < 500
			innerHas4xx := strings.Contains(body, "HTTP/1.1 4")

			if !outerIs4xx && !innerHas4xx {
				return framework.NewError(
					"spec violation (§11.4.9.2): GET inside a changeset must be rejected with 4xx, " +
						"but the server returned a success response")
			}
			return nil
		},
	)

	return suite
}
