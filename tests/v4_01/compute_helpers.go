package v4_01

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/nlstn/odata-compliance-suite/framework"
)

type collectionResponse struct {
	Value []map[string]interface{} `json:"value"`
}

func requireStatusOK(resp *framework.HTTPResponse) error {
	if resp.StatusCode != http.StatusOK {
		return framework.NewError(fmt.Sprintf("expected HTTP 200 OK but got %d", resp.StatusCode))
	}
	return nil
}

func decodeCollection(resp *framework.HTTPResponse) ([]map[string]interface{}, error) {
	var payload collectionResponse
	if err := json.Unmarshal(resp.Body, &payload); err != nil {
		return nil, framework.NewError(fmt.Sprintf("failed to parse JSON response: %v", err))
	}

	if payload.Value == nil {
		return nil, framework.NewError("response body does not contain 'value' collection")
	}

	if len(payload.Value) == 0 {
		return nil, framework.NewError("response collection is empty; cannot validate computed properties")
	}

	return payload.Value, nil
}

func ensureComputedProperties(entities []map[string]interface{}, aliases ...string) error {
	for i, entity := range entities {
		for _, alias := range aliases {
			if _, ok := entity[alias]; !ok {
				return framework.NewError(fmt.Sprintf("entity %d missing computed property %q", i, alias))
			}
		}
	}

	return nil
}

// assertComputedSortOrder verifies that entities are sorted by the named
// computed field (which must be a JSON number). Requires at least 2 entities.
func assertComputedSortOrder(entities []map[string]interface{}, field string, ascending bool) error {
	if len(entities) < 2 {
		return nil // nothing to compare
	}
	for i := 1; i < len(entities); i++ {
		prev, err := entityFloat(entities[i-1], field)
		if err != nil {
			return fmt.Errorf("entity %d field %q: %w", i-1, field, err)
		}
		curr, err := entityFloat(entities[i], field)
		if err != nil {
			return fmt.Errorf("entity %d field %q: %w", i, field, err)
		}
		if ascending && prev > curr {
			return framework.NewError(fmt.Sprintf(
				"sort order violated at index %d: %v > %v (expected ascending by %q)",
				i, prev, curr, field))
		}
		if !ascending && prev < curr {
			return framework.NewError(fmt.Sprintf(
				"sort order violated at index %d: %v < %v (expected descending by %q)",
				i, prev, curr, field))
		}
	}
	return nil
}

func entityFloat(entity map[string]interface{}, field string) (float64, error) {
	raw, ok := entity[field]
	if !ok {
		return 0, fmt.Errorf("field absent")
	}
	switch v := raw.(type) {
	case float64:
		return v, nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	default:
		return 0, fmt.Errorf("unexpected type %T", raw)
	}
}
