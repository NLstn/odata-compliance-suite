package framework_test

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// withTestContext runs fn with a live *framework.TestContext. TestContext has
// no exported constructor, so obtaining one means driving a minimal suite
// through Run() and capturing ctx from inside a test body, exactly as a real
// compliance test would receive it.
func withTestContext(t *testing.T, fn func(ctx *framework.TestContext)) {
	t.Helper()
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{}`)),
			Request:    req,
		}, nil
	})}

	suite := framework.NewTestSuite("assert-helpers", "assert-helpers", "https://example.test/spec")
	suite.ServerURL = "http://example.test"
	suite.Client = client
	suite.Out = io.Discard
	suite.Quiet = true
	suite.AddTest("check", "check", func(ctx *framework.TestContext) error {
		fn(ctx)
		return nil
	})

	if err := suite.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
}

func TestAssertStatusCode(t *testing.T) {
	withTestContext(t, func(ctx *framework.TestContext) {
		if err := ctx.AssertStatusCode(&framework.HTTPResponse{StatusCode: 200}, 200); err != nil {
			t.Errorf("matching status: got error %v, want nil", err)
		}

		err := ctx.AssertStatusCode(&framework.HTTPResponse{StatusCode: 404, Body: []byte(`{"error":"not found"}`)}, 200)
		if err == nil {
			t.Fatal("mismatched status: got nil error, want error")
		}
		if !strings.Contains(err.Error(), "404") || !strings.Contains(err.Error(), "200") {
			t.Errorf("error should mention both actual and expected status, got: %v", err)
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("error should include body preview, got: %v", err)
		}

		longBody := strings.Repeat("x", 300)
		err = ctx.AssertStatusCode(&framework.HTTPResponse{StatusCode: 500, Body: []byte(longBody)}, 200)
		if err == nil {
			t.Fatal("expected error for long body case")
		}
		if strings.Contains(err.Error(), longBody) {
			t.Error("body preview should be truncated, not include the full 300-char body")
		}
		if !strings.Contains(err.Error(), "...") {
			t.Error("truncated body preview should be marked with an ellipsis")
		}
	})
}

func TestAssertODataError(t *testing.T) {
	withTestContext(t, func(ctx *framework.TestContext) {
		validBody := func(message string) []byte {
			return []byte(`{"error":{"code":"EntityNotFound","message":"` + message + `"}}`)
		}

		if err := ctx.AssertODataError(&framework.HTTPResponse{StatusCode: 404, Body: validBody("no such entity")}, 404, ""); err != nil {
			t.Errorf("valid error body, empty fragment: got %v, want nil", err)
		}

		if err := ctx.AssertODataError(&framework.HTTPResponse{StatusCode: 404, Body: validBody("No Such Entity")}, 404, "such entity"); err != nil {
			t.Errorf("fragment should match case-insensitively: got %v", err)
		}

		if err := ctx.AssertODataError(&framework.HTTPResponse{StatusCode: 200, Body: validBody("no such entity")}, 404, ""); err == nil {
			t.Error("status mismatch should error even with a well-formed error body")
		}

		if err := ctx.AssertODataError(&framework.HTTPResponse{StatusCode: 404, Body: []byte(`{"foo":"bar"}`)}, 404, ""); err == nil {
			t.Error("missing error object should error")
		} else if !strings.Contains(err.Error(), "missing error object") {
			t.Errorf("unexpected error message: %v", err)
		}

		if err := ctx.AssertODataError(&framework.HTTPResponse{StatusCode: 404, Body: []byte(`{"error":"not an object"}`)}, 404, ""); err == nil {
			t.Error("error value of unexpected type should error")
		}

		if err := ctx.AssertODataError(&framework.HTTPResponse{StatusCode: 404, Body: []byte(`{"error":{"message":"no code here"}}`)}, 404, ""); err == nil {
			t.Error("missing code should error")
		}

		if err := ctx.AssertODataError(&framework.HTTPResponse{StatusCode: 404, Body: []byte(`{"error":{"code":"","message":"blank code"}}`)}, 404, ""); err == nil {
			t.Error("empty code should error")
		}

		messageObjBody := []byte(`{"error":{"code":"E1","message":{"lang":"en","value":"wrapped message"}}}`)
		if err := ctx.AssertODataError(&framework.HTTPResponse{StatusCode: 400, Body: messageObjBody}, 400, "wrapped message"); err != nil {
			t.Errorf("message-as-object form should be extracted: got %v", err)
		}

		detailBody := []byte(`{"error":{"code":"E1","message":"top level","details":[{"message":"nested detail text"}]}}`)
		if err := ctx.AssertODataError(&framework.HTTPResponse{StatusCode: 400, Body: detailBody}, 400, "nested detail text"); err != nil {
			t.Errorf("fragment matching a details[] message should be accepted: got %v", err)
		}

		if err := ctx.AssertODataError(&framework.HTTPResponse{StatusCode: 404, Body: validBody("no such entity")}, 404, "totally unrelated text"); err == nil {
			t.Error("non-matching fragment should error")
		}
	})
}

func TestAssertHeader(t *testing.T) {
	withTestContext(t, func(ctx *framework.TestContext) {
		h := make(http.Header)
		h.Set("OData-Version", "4.0")
		resp := &framework.HTTPResponse{Headers: h}

		if err := ctx.AssertHeader(resp, "OData-Version", "4.0"); err != nil {
			t.Errorf("matching header: got %v, want nil", err)
		}
		if err := ctx.AssertHeader(resp, "OData-Version", "4.01"); err == nil {
			t.Error("mismatched header value should error")
		}
		if err := ctx.AssertHeader(resp, "X-Missing", ""); err != nil {
			t.Errorf("missing header vs expected empty string should be treated as a match: got %v", err)
		}
		if err := ctx.AssertHeader(resp, "X-Missing", "present"); err == nil {
			t.Error("missing header vs non-empty expected value should error")
		}
	})
}

func TestAssertHeaderContains(t *testing.T) {
	withTestContext(t, func(ctx *framework.TestContext) {
		h := make(http.Header)
		h.Set("Content-Type", "application/json;odata.metadata=minimal")
		resp := &framework.HTTPResponse{Headers: h}

		if err := ctx.AssertHeaderContains(resp, "Content-Type", "odata.metadata=minimal"); err != nil {
			t.Errorf("substring present: got %v, want nil", err)
		}
		if err := ctx.AssertHeaderContains(resp, "Content-Type", "odata.metadata=full"); err == nil {
			t.Error("substring absent should error")
		}
	})
}

func TestAssertJSONField(t *testing.T) {
	withTestContext(t, func(ctx *framework.TestContext) {
		resp := &framework.HTTPResponse{Body: []byte(`{"Name":"Widget","Notes":null}`)}

		if err := ctx.AssertJSONField(resp, "Name"); err != nil {
			t.Errorf("present field: got %v, want nil", err)
		}
		if err := ctx.AssertJSONField(resp, "Notes"); err != nil {
			t.Errorf("field present with JSON null value should still count as present: got %v", err)
		}
		if err := ctx.AssertJSONField(resp, "Missing"); err == nil {
			t.Error("absent field should error")
		}
		if err := ctx.AssertJSONField(&framework.HTTPResponse{Body: []byte(`not json`)}, "Name"); err == nil {
			t.Error("malformed JSON should error")
		}
	})
}

func TestAssertBodyContains(t *testing.T) {
	withTestContext(t, func(ctx *framework.TestContext) {
		resp := &framework.HTTPResponse{Body: []byte(`{"@odata.context":"http://x/$metadata#Products"}`)}
		if err := ctx.AssertBodyContains(resp, "@odata.context"); err != nil {
			t.Errorf("substring present: got %v, want nil", err)
		}
		if err := ctx.AssertBodyContains(resp, "@odata.count"); err == nil {
			t.Error("substring absent should error")
		}
	})
}

func TestAssertEntityHasFields(t *testing.T) {
	withTestContext(t, func(ctx *framework.TestContext) {
		entity := map[string]interface{}{"ID": "1", "Name": "Widget"}

		if err := ctx.AssertEntityHasFields(entity, "ID", "Name"); err != nil {
			t.Errorf("all present: got %v, want nil", err)
		}
		if err := ctx.AssertEntityHasFields(entity, "ID", "Price"); err == nil {
			t.Error("one missing field should error")
		} else if !strings.Contains(err.Error(), "Price") {
			t.Errorf("error should name the missing field, got: %v", err)
		}
	})
}

func TestAssertEntityOnlyAllowedFieldsEmptyAllowList(t *testing.T) {
	withTestContext(t, func(ctx *framework.TestContext) {
		if err := ctx.AssertEntityOnlyAllowedFields(map[string]interface{}{"@odata.id": "x"}); err != nil {
			t.Errorf("empty allow-list with only annotations should pass: got %v", err)
		}
		if err := ctx.AssertEntityOnlyAllowedFields(map[string]interface{}{"ID": "1"}); err == nil {
			t.Error("empty allow-list with a real structural field should error")
		}
	})
}

func TestAssertAllEntitiesSatisfy(t *testing.T) {
	withTestContext(t, func(ctx *framework.TestContext) {
		items := []map[string]interface{}{
			{"Price": 10.0},
			{"Price": 20.0},
		}

		err := ctx.AssertAllEntitiesSatisfy(items, "Price > 0", func(e map[string]interface{}) (bool, string) {
			return e["Price"].(float64) > 0, ""
		})
		if err != nil {
			t.Errorf("all satisfy: got %v, want nil", err)
		}

		err = ctx.AssertAllEntitiesSatisfy(items, "Price > 15", func(e map[string]interface{}) (bool, string) {
			return e["Price"].(float64) > 15, ""
		})
		if err == nil {
			t.Fatal("one violating entity should error")
		}
		if !strings.Contains(err.Error(), "index 0") {
			t.Errorf("error should name the violating index, got: %v", err)
		}
		if !strings.Contains(err.Error(), "predicate returned false") {
			t.Errorf("empty reason should default to a generic message, got: %v", err)
		}

		if err := ctx.AssertAllEntitiesSatisfy(nil, "anything", func(map[string]interface{}) (bool, string) {
			t.Fatal("predicate should not be invoked for an empty collection")
			return false, ""
		}); err != nil {
			t.Errorf("empty collection is vacuously satisfied: got %v", err)
		}
	})
}

func TestAssertEntitiesSortedByFloat(t *testing.T) {
	withTestContext(t, func(ctx *framework.TestContext) {
		asc := []map[string]interface{}{{"Price": 1.0}, {"Price": 2.0}, {"Price": 3.0}}
		if err := ctx.AssertEntitiesSortedByFloat(asc, "Price", true); err != nil {
			t.Errorf("ascending data, ascending check: got %v, want nil", err)
		}
		if err := ctx.AssertEntitiesSortedByFloat(asc, "Price", false); err == nil {
			t.Error("ascending data checked as descending should error")
		}

		desc := []map[string]interface{}{{"Price": 3.0}, {"Price": 2.0}, {"Price": 1.0}}
		if err := ctx.AssertEntitiesSortedByFloat(desc, "Price", false); err != nil {
			t.Errorf("descending data, descending check: got %v, want nil", err)
		}

		tied := []map[string]interface{}{{"Price": 1.0}, {"Price": 1.0}, {"Price": 1.0}}
		if err := ctx.AssertEntitiesSortedByFloat(tied, "Price", true); err != nil {
			t.Errorf("tied values should satisfy ascending: got %v", err)
		}
		if err := ctx.AssertEntitiesSortedByFloat(tied, "Price", false); err != nil {
			t.Errorf("tied values should satisfy descending: got %v", err)
		}

		if err := ctx.AssertEntitiesSortedByFloat([]map[string]interface{}{{"Price": 1.0}}, "Price", true); err != nil {
			t.Errorf("fewer than 2 items should trivially pass: got %v", err)
		}

		missingField := []map[string]interface{}{{"Price": 1.0}, {"Other": 2.0}}
		if err := ctx.AssertEntitiesSortedByFloat(missingField, "Price", true); err == nil {
			t.Error("missing field on one item should error")
		}

		// The OData JSON format represents Edm.Double NaN/INF as the strings
		// "NaN"/"INF", not a numeric literal, so a genuine service response
		// decodes to a string here, not a float64 — confirm that surfaces as
		// a clear "not numeric" error rather than silently comparing false
		// (which would mask any actual ordering bug involving these values).
		nonNumeric := []map[string]interface{}{{"Price": "NaN"}, {"Price": 1.0}}
		if err := ctx.AssertEntitiesSortedByFloat(nonNumeric, "Price", true); err == nil {
			t.Error("non-numeric field value should error, not be silently treated as unordered-but-passing")
		}
	})
}
