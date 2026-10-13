package v4_0

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// BatchContentIDReferencing creates the 11.4.9.3 Referencing New Entities compliance
// test suite.
//
// OData v4 spec §11.4.9.3: within a changeset, a request bearing a Content-ID header
// creates an alias "$<contentID>" that subsequent requests in the same changeset may use
// as a URL prefix to refer to the newly created entity.
func BatchContentIDReferencing() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.4.9.3 Referencing New Entities in a Change Set",
		"Tests $<contentID> URL referencing within $batch changesets per OData v4 spec §11.4.9.3.",
		"https://docs.oasis-open.org/odata/odata/v4.01/odata-v4.01-part1-protocol.html#sec_ReferencingNewEntities",
	)

	// Helper: return a POST body for a minimal Product.
	postProduct := func(name string) string {
		return fmt.Sprintf(`{"Name":%q,"Price":10.0,"Status":1}`, name)
	}

	// productHeaders are included in batch sub-requests that create Products.
	// The devserver's ODataBeforeCreate hook for Product requires X-User-Role: admin.
	const productHeaders = "Content-Type: application/json\nX-User-Role: admin\n"

	// -----------------------------------------------------------------------
	// Test 1: create entity, write via $<contentID>, then read after commit
	// -----------------------------------------------------------------------
	suite.AddTest(
		"test_content_id_read_back",
		"Entity updated via $<contentID> can be read after the changeset commits",
		func(ctx *framework.TestContext) error {
			batchBoundary := "batch_cid_rb"
			changesetBoundary := "changeset_cid_rb"
			body := fmt.Sprintf(`--%s
Content-Type: multipart/mixed; boundary=%s

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: 1

POST /Products HTTP/1.1
Content-Type: application/json
X-User-Role: admin

%s

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: 2

PATCH /$1 HTTP/1.1
Content-Type: application/json

{"Name":"CID ReadBack Updated"}

--%s--

--%s--`,
				batchBoundary, changesetBoundary,
				changesetBoundary, postProduct("CID ReadBack Product"),
				changesetBoundary, changesetBoundary,
				batchBoundary)

			resp, err := ctx.POSTRaw("/$batch", []byte(body),
				fmt.Sprintf("multipart/mixed; boundary=%s", batchBoundary))
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			respBody := string(resp.Body)
			if !strings.Contains(respBody, "HTTP/1.1 201") {
				return framework.NewError("expected 201 Created for POST with Content-ID")
			}
			if !strings.Contains(respBody, "HTTP/1.1 200") && !strings.Contains(respBody, "HTTP/1.1 204") {
				return framework.NewError("expected 200 or 204 for PATCH via $1")
			}
			_, err = contentIDProduct(ctx, "CID ReadBack Updated", false)
			return err
		},
	)

	// -----------------------------------------------------------------------
	// Test 2: create entity, update via $<contentID>
	// -----------------------------------------------------------------------
	suite.AddTest(
		"test_content_id_patch",
		"Created entity can be updated via $<contentID> PATCH in same changeset",
		func(ctx *framework.TestContext) error {
			batchBoundary := "batch_cid_patch"
			changesetBoundary := "changeset_cid_patch"
			body := fmt.Sprintf(`--%s
Content-Type: multipart/mixed; boundary=%s

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: 1

POST /Products HTTP/1.1
Content-Type: application/json
X-User-Role: admin

%s

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: 2

PATCH /$1 HTTP/1.1
Content-Type: application/json
X-User-Role: admin

{"Name":"CID Patched Product","Price":20.0,"Status":1}

--%s--

--%s--`,
				batchBoundary, changesetBoundary,
				changesetBoundary, postProduct("CID Original Product"),
				changesetBoundary, changesetBoundary,
				batchBoundary)

			resp, err := ctx.POSTRaw("/$batch", []byte(body),
				fmt.Sprintf("multipart/mixed; boundary=%s", batchBoundary))
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			respBody := string(resp.Body)
			if !strings.Contains(respBody, "HTTP/1.1 201") {
				return framework.NewError("expected 201 Created for POST with Content-ID")
			}
			// PATCH returns 200 or 204.
			if !strings.Contains(respBody, "HTTP/1.1 200") &&
				!strings.Contains(respBody, "HTTP/1.1 204") {
				return framework.NewError("expected 200 or 204 for PATCH via $1")
			}
			return nil
		},
	)

	// -----------------------------------------------------------------------
	// Test 3: create entity, delete via $<contentID>
	// -----------------------------------------------------------------------
	suite.AddTest(
		"test_content_id_delete",
		"Created entity can be deleted via $<contentID> DELETE in same changeset",
		func(ctx *framework.TestContext) error {
			batchBoundary := "batch_cid_del"
			changesetBoundary := "changeset_cid_del"
			body := fmt.Sprintf(`--%s
Content-Type: multipart/mixed; boundary=%s

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: 1

POST /Products HTTP/1.1
Content-Type: application/json
X-User-Role: admin

%s

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: 2

DELETE /$1 HTTP/1.1


--%s--

--%s--`,
				batchBoundary, changesetBoundary,
				changesetBoundary, postProduct("CID Delete Product"),
				changesetBoundary, changesetBoundary,
				batchBoundary)

			resp, err := ctx.POSTRaw("/$batch", []byte(body),
				fmt.Sprintf("multipart/mixed; boundary=%s", batchBoundary))
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			respBody := string(resp.Body)
			if !strings.Contains(respBody, "HTTP/1.1 201") {
				return framework.NewError("expected 201 Created for POST with Content-ID")
			}
			if !strings.Contains(respBody, "HTTP/1.1 204") {
				return framework.NewError("expected 204 No Content for DELETE via $1")
			}
			return nil
		},
	)

	// -----------------------------------------------------------------------
	// Test 4: multiple independent Content-IDs in one changeset
	// -----------------------------------------------------------------------
	suite.AddTest(
		"test_content_id_multiple",
		"Multiple Content-IDs can coexist and be independently resolved",
		func(ctx *framework.TestContext) error {
			batchBoundary := "batch_cid_multi"
			changesetBoundary := "changeset_cid_multi"
			body := fmt.Sprintf(`--%s
Content-Type: multipart/mixed; boundary=%s

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: 1

POST /Products HTTP/1.1
Content-Type: application/json
X-User-Role: admin

%s

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: 2

POST /Products HTTP/1.1
Content-Type: application/json
X-User-Role: admin

%s

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: 3

PATCH /$1 HTTP/1.1
Content-Type: application/json
X-User-Role: admin

{"Name":"Multi CID Alpha Updated","Price":10.0,"Status":1}

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: 4

PATCH /$2 HTTP/1.1
Content-Type: application/json
X-User-Role: admin

{"Name":"Multi CID Beta Updated","Price":10.0,"Status":1}

--%s--

--%s--`,
				batchBoundary, changesetBoundary,
				changesetBoundary, postProduct("Multi CID Alpha"),
				changesetBoundary, postProduct("Multi CID Beta"),
				changesetBoundary, changesetBoundary, changesetBoundary,
				batchBoundary)

			resp, err := ctx.POSTRaw("/$batch", []byte(body),
				fmt.Sprintf("multipart/mixed; boundary=%s", batchBoundary))
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			respBody := string(resp.Body)
			createdCount := strings.Count(respBody, "HTTP/1.1 201")
			if createdCount != 2 {
				return framework.NewError(
					fmt.Sprintf("expected 2 × 201 Created, got %d", createdCount))
			}
			// Two PATCH responses (200 or 204 each).
			patchSuccess := strings.Count(respBody, "HTTP/1.1 200") +
				strings.Count(respBody, "HTTP/1.1 204")
			if patchSuccess < 2 {
				return framework.NewError(
					fmt.Sprintf("expected 2 successful PATCH responses, got %d", patchSuccess))
			}
			return nil
		},
	)

	// -----------------------------------------------------------------------
	// Test 5: $<contentID> in navigation-property URL segment
	// -----------------------------------------------------------------------
	suite.AddTest(
		"test_content_id_navigation_property",
		"$<contentID> supports navigation-property writes verified after commit",
		func(ctx *framework.TestContext) error {
			batchBoundary := "batch_cid_nav"
			changesetBoundary := "changeset_cid_nav"
			body := fmt.Sprintf(`--%s
Content-Type: multipart/mixed; boundary=%s

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: 1

POST /Products HTTP/1.1
Content-Type: application/json
X-User-Role: admin

%s

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: 2

POST /$1/Descriptions HTTP/1.1
Content-Type: application/json

{"LanguageKey":"EN","Description":"CID navigation description"}

--%s--

--%s--`,
				batchBoundary, changesetBoundary,
				changesetBoundary, postProduct("CID Nav Product"),
				changesetBoundary,
				changesetBoundary,
				batchBoundary)

			resp, err := ctx.POSTRaw("/$batch", []byte(body),
				fmt.Sprintf("multipart/mixed; boundary=%s", batchBoundary))
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			respBody := string(resp.Body)
			if !strings.Contains(respBody, "HTTP/1.1 201") {
				return framework.NewError("expected 201 Created for POST with Content-ID")
			}
			if strings.Count(respBody, "HTTP/1.1 201") != 2 {
				return framework.NewError("expected 201 Created for both product and navigation POST")
			}
			product, err := contentIDProduct(ctx, "CID Nav Product", true)
			if err != nil {
				return err
			}
			descriptions, ok := product["Descriptions"].([]interface{})
			if !ok || len(descriptions) != 1 {
				return fmt.Errorf("expected one persisted description, got %v", product["Descriptions"])
			}
			description, ok := descriptions[0].(map[string]interface{})
			if !ok || description["LanguageKey"] != "EN" || description["Description"] != "CID navigation description" {
				return fmt.Errorf("navigation write did not persist expected description: %v", descriptions[0])
			}
			return nil
		},
	)

	// -----------------------------------------------------------------------
	// Test 6: Content-ID response echo
	// -----------------------------------------------------------------------
	suite.AddTest(
		"test_content_id_echo_in_response",
		"Content-ID is echoed back in every response MIME part (including $<contentID> requests)",
		func(ctx *framework.TestContext) error {
			batchBoundary := "batch_cid_echo"
			changesetBoundary := "changeset_cid_echo"
			body := fmt.Sprintf(`--%s
Content-Type: multipart/mixed; boundary=%s

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: create-req

POST /Products HTTP/1.1
Content-Type: application/json
X-User-Role: admin

%s

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: patch-req

PATCH /$create-req HTTP/1.1
Content-Type: application/json

{"Name":"CID Echo Patched","Price":10.0,"Status":1}

--%s--

--%s--`,
				batchBoundary, changesetBoundary,
				changesetBoundary, postProduct("CID Echo Product"),
				changesetBoundary, changesetBoundary,
				batchBoundary)

			resp, err := ctx.POSTRaw("/$batch", []byte(body),
				fmt.Sprintf("multipart/mixed; boundary=%s", batchBoundary))
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			respBody := string(resp.Body)
			if !strings.Contains(respBody, "Content-ID: create-req") {
				return framework.NewError("response missing Content-ID: create-req")
			}
			if !strings.Contains(respBody, "Content-ID: patch-req") {
				return framework.NewError("response missing Content-ID: patch-req")
			}
			return nil
		},
	)

	// -----------------------------------------------------------------------
	// Test 7: unresolvable $<contentID> causes failure (and changeset rollback)
	// -----------------------------------------------------------------------
	suite.AddTest(
		"test_content_id_unresolvable_causes_error",
		"Unresolvable $<contentID> reference causes request failure and changeset rollback",
		func(ctx *framework.TestContext) error {
			batchBoundary := "batch_cid_unres"
			changesetBoundary := "changeset_cid_unres"
			body := fmt.Sprintf(`--%s
Content-Type: multipart/mixed; boundary=%s

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: 1

POST /Products HTTP/1.1
Content-Type: application/json
X-User-Role: admin

%s

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: 2

PATCH /$99 HTTP/1.1
Content-Type: application/json

{"Name":"Should not apply","Price":10.0,"Status":1}

--%s--

--%s--`,
				batchBoundary, changesetBoundary,
				changesetBoundary, postProduct("CID Unres Product"),
				changesetBoundary, changesetBoundary,
				batchBoundary)

			resp, err := ctx.POSTRaw("/$batch", []byte(body),
				fmt.Sprintf("multipart/mixed; boundary=%s", batchBoundary))
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			if err := assertFailedContentIDChangeset(ctx, resp); err != nil {
				return err
			}
			return assertNoProductNamed(ctx, "CID Unres Product")
		},
	)

	// -----------------------------------------------------------------------
	// Test 8: duplicate Content-ID within the same changeset must be rejected
	// -----------------------------------------------------------------------
	suite.AddTest(
		"test_content_id_duplicate_rejected",
		"Duplicate Content-ID within the same changeset is rejected, not silently accepted",
		func(ctx *framework.TestContext) error {
			batchBoundary := "batch_cid_dup"
			changesetBoundary := "changeset_cid_dup"
			body := fmt.Sprintf(`--%s
Content-Type: multipart/mixed; boundary=%s

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: 1

POST /Products HTTP/1.1
Content-Type: application/json
X-User-Role: admin

%s

--%s
Content-Type: application/http
Content-Transfer-Encoding: binary
Content-ID: 1

POST /Products HTTP/1.1
Content-Type: application/json
X-User-Role: admin

%s

--%s--

--%s--`,
				batchBoundary, changesetBoundary,
				changesetBoundary, postProduct("CID Duplicate A"),
				changesetBoundary, postProduct("CID Duplicate B"),
				changesetBoundary,
				batchBoundary)

			resp, err := ctx.POSTRaw("/$batch", []byte(body),
				fmt.Sprintf("multipart/mixed; boundary=%s", batchBoundary))
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			respBody := string(resp.Body)
			createdCount := strings.Count(respBody, "HTTP/1.1 201")
			if createdCount == 2 {
				return framework.NewError(
					"duplicate Content-ID within a changeset was accepted (both sub-requests returned 201) instead of being rejected; see NLstn/go-odata#815")
			}
			if !strings.Contains(respBody, "HTTP/1.1 4") {
				return fmt.Errorf("expected duplicate Content-ID to be rejected with a 4xx sub-response, got: %s", respBody)
			}

			// The changeset is atomic: a rejected duplicate must roll back any
			// entity the changeset already created, not leave it persisted.
			verifyResp, err := ctx.GET("/Products?$filter=" + url.QueryEscape("contains(Name,'CID Duplicate')"))
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(verifyResp, 200); err != nil {
				return err
			}
			items, err := ctx.ParseEntityCollection(verifyResp)
			if err != nil {
				return err
			}
			if len(items) != 0 {
				return fmt.Errorf("changeset should have rolled back after the duplicate Content-ID was rejected, but %d matching product(s) still exist", len(items))
			}
			return nil
		},
	)

	return suite
}

// A failed changeset is one application/http part, not a nested multipart
// containing successful operations that were rolled back (§11.4.9.4).
func assertFailedContentIDChangeset(ctx *framework.TestContext, resp *framework.HTTPResponse) error {
	mediaType, params, err := mime.ParseMediaType(resp.Headers.Get("Content-Type"))
	if err != nil {
		return err
	}
	if mediaType != "multipart/mixed" || params["boundary"] == "" {
		return fmt.Errorf("expected multipart batch response, got %q", resp.Headers.Get("Content-Type"))
	}
	reader := multipart.NewReader(bytes.NewReader(resp.Body), params["boundary"])
	part, err := reader.NextPart()
	if err != nil {
		return fmt.Errorf("missing changeset error response: %w", err)
	}
	partType, _, err := mime.ParseMediaType(part.Header.Get("Content-Type"))
	if err != nil {
		return err
	}
	if partType != "application/http" {
		return fmt.Errorf("failed changeset must be a single application/http error, got %q", partType)
	}
	inner, err := http.ReadResponse(bufio.NewReader(part), nil)
	if err != nil {
		return fmt.Errorf("invalid changeset HTTP response: %w", err)
	}
	defer inner.Body.Close()
	body, err := io.ReadAll(inner.Body)
	if err != nil {
		return err
	}
	if inner.StatusCode < 400 || inner.StatusCode >= 500 {
		return fmt.Errorf("expected 4xx for unresolvable $99, got %d", inner.StatusCode)
	}
	if err := ctx.AssertODataError(&framework.HTTPResponse{StatusCode: inner.StatusCode, Headers: inner.Header, Body: body}, inner.StatusCode, ""); err != nil {
		return err
	}
	if _, err := reader.NextPart(); err != io.EOF {
		return fmt.Errorf("expected exactly one changeset error response, got another part or parsing error: %v", err)
	}
	return nil
}

func contentIDProduct(ctx *framework.TestContext, name string, expand bool) (map[string]interface{}, error) {
	query := url.Values{"$filter": {"Name eq '" + strings.ReplaceAll(name, "'", "''") + "'"}}
	if expand {
		query.Set("$expand", "Descriptions")
	}
	resp, err := ctx.GET("/Products?" + query.Encode())
	if err != nil {
		return nil, err
	}
	if err := ctx.AssertStatusCode(resp, 200); err != nil {
		return nil, err
	}
	items, err := ctx.ParseEntityCollection(resp)
	if err != nil {
		return nil, err
	}
	if len(items) != 1 {
		return nil, fmt.Errorf("expected one persisted product named %q, got %d", name, len(items))
	}
	return items[0], nil
}
