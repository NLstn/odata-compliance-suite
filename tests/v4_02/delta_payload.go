package v4_02

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"slices"
	"strings"

	"github.com/nlstn/odata-compliance-suite/framework"
)

func DeltaPayload() *framework.TestSuite {
	suite := framework.NewTestSuite("15 OData 4.02 delta payload", "Checks the ETag of an added entity and ordered control information on a deletion.", jsonSpec+"#sec_DeletedEntity")
	suite.AddTestWithCapabilities("test_delta_etag_and_deleted_type_order",
		"Delta entries carry the added entity ETag and keep deleted type immediately after removed",
		[]framework.RequiredCapability{
			framework.Require(framework.CapInsert, "Products"),
			framework.Require(framework.CapDelete, "Products"),
		}, testDeltaPayload)
	return suite
}

func testDeltaPayload(ctx *framework.TestContext) error {
	full := framework.Header{Key: "Accept", Value: "application/json;metadata=full;streaming=true"}
	initial, err := ctx.GET("/Products", max402, framework.Header{Key: "Accept", Value: "application/json"},
		framework.Header{Key: "Prefer", Value: "track-changes"})
	if err != nil {
		return err
	}
	if err := assertResponse(ctx, initial, http.StatusOK, "4.02"); err != nil {
		return err
	}
	_, next, err := decodeDelta(ctx, initial)
	if err != nil {
		return err
	}

	created, err := ctx.POST("/Products", map[string]interface{}{
		"Name": "OData 4.02 delta fixture", "Price": 1.25, "Status": 1,
	}, max402, payload402)
	if err != nil {
		return err
	}
	if err := assertResponse(ctx, created, http.StatusCreated, "4.02"); err != nil {
		return err
	}
	var product struct {
		ID string `json:"ID"`
	}
	if err := ctx.GetJSON(created, &product); err != nil {
		return err
	}
	if product.ID == "" {
		return fmt.Errorf("created delta fixture has no ID")
	}
	path := fmt.Sprintf("/Products(%s)", product.ID)
	deleted := false
	defer func() {
		if !deleted {
			if _, err := ctx.DELETE(path, max402, framework.Header{Key: "If-Match", Value: "*"}); err != nil {
				ctx.Log(fmt.Sprintf("delta fixture cleanup failed: %v", err))
			}
		}
	}()
	tag := created.Headers.Get("ETag")
	if tag == "" {
		return fmt.Errorf("created entity has no ETag")
	}
	added, err := ctx.GETNextLink(next, max402, full)
	if err != nil {
		return err
	}
	if err := assertResponse(ctx, added, http.StatusOK, "4.02"); err != nil {
		return err
	}
	entries, next, err := decodeDelta(ctx, added)
	if err != nil {
		return err
	}
	if len(entries) != 1 {
		return fmt.Errorf("expected one added entity, got %d entries", len(entries))
	}
	var entry map[string]interface{}
	if err := json.Unmarshal(entries[0], &entry); err != nil {
		return err
	}
	if entry["ID"] != product.ID || control(entry, "etag") != tag {
		return fmt.Errorf("delta added entity ID or ETag does not match the create response: %v", entry)
	}

	removed, err := ctx.DELETE(path, max402, framework.Header{Key: "If-Match", Value: tag})
	if err != nil {
		return err
	}
	if err := assertResponse(ctx, removed, http.StatusNoContent, "4.02"); err != nil {
		return err
	}
	deleted = true
	delta, err := ctx.GETNextLink(next, max402, full)
	if err != nil {
		return err
	}
	if err := assertResponse(ctx, delta, http.StatusOK, "4.02"); err != nil {
		return err
	}
	_, mediaParams, err := mime.ParseMediaType(delta.Headers.Get("Content-Type"))
	if err != nil || mediaParams["streaming"] != "true" {
		return fmt.Errorf("delta Content-Type must advertise streaming=true, got %q", delta.Headers.Get("Content-Type"))
	}

	entries, _, err = decodeDelta(ctx, delta)
	if err != nil {
		return err
	}
	if len(entries) != 1 {
		return fmt.Errorf("expected one deleted entity, got %d entries", len(entries))
	}

	fields, err := orderedFields(entries[0])
	if err != nil {
		return err
	}
	if len(fields) < 2 || fields[0] != "context" || fields[1] != "removed" {
		return fmt.Errorf("deleted entity must start with context and removed, got %v", fields)
	}
	if index := slices.Index(fields, "type"); index >= 0 && index != 2 {
		return fmt.Errorf("deleted entity type must immediately follow removed, got %v", fields)
	}
	var deletedEntry map[string]interface{}
	if err := json.Unmarshal(entries[0], &deletedEntry); err != nil {
		return err
	}
	removal, ok := control(deletedEntry, "removed").(map[string]interface{})
	if !ok || removal["reason"] != "deleted" || deletedEntry["ID"] != product.ID {
		return fmt.Errorf("unexpected deleted entity: %v", deletedEntry)
	}
	return nil
}

func control(entry map[string]interface{}, name string) interface{} {
	value, ok := entry["@"+name]
	if ok {
		return value
	}
	return entry["@odata."+name]
}

func decodeDelta(ctx *framework.TestContext, resp *framework.HTTPResponse) ([]json.RawMessage, string, error) {
	var envelope struct {
		Value      []json.RawMessage `json:"value"`
		DeltaLink  string            `json:"@deltaLink"`
		LegacyLink string            `json:"@odata.deltaLink"`
	}
	if err := ctx.GetJSON(resp, &envelope); err != nil {
		return nil, "", err
	}
	next := envelope.DeltaLink
	if next == "" {
		next = envelope.LegacyLink
	}
	if next == "" || envelope.Value == nil {
		return nil, "", fmt.Errorf("delta response is missing value or deltaLink")
	}
	return envelope.Value, next, nil
}

func orderedFields(raw json.RawMessage) ([]string, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	token, err := decoder.Token()
	if err != nil || token != json.Delim('{') {
		return nil, fmt.Errorf("expected a JSON object")
	}
	var fields []string
	for decoder.More() {
		name, err := decoder.Token()
		if err != nil {
			return nil, err
		}
		field, ok := name.(string)
		if !ok {
			return nil, fmt.Errorf("expected a JSON member name")
		}
		fields = append(fields, strings.TrimPrefix(strings.TrimPrefix(field, "@"), "odata."))
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return nil, err
		}
	}
	return fields, nil
}
