package v4_01

import (
	"encoding/json"
	"fmt"

	"github.com/nlstn/odata-compliance-suite/framework"
)

func firstEntityPath(ctx *framework.TestContext, entitySet string) (string, error) {
	id, err := firstEntityID(ctx, entitySet)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("/%s(%s)", entitySet, id), nil
}

func firstEntityID(ctx *framework.TestContext, entitySet string) (string, error) {
	resp, err := ctx.GET(fmt.Sprintf("/%s?$top=1&$select=ID", entitySet))
	if err != nil {
		return "", err
	}
	if err := ctx.AssertStatusCode(resp, 200); err != nil {
		return "", fmt.Errorf("list %s: %w", entitySet, err)
	}

	var body struct {
		Value []map[string]interface{} `json:"value"`
	}
	if err := json.Unmarshal(resp.Body, &body); err != nil {
		return "", fmt.Errorf("parse %s list: %w", entitySet, err)
	}
	if len(body.Value) == 0 {
		return "", fmt.Errorf("no entities in %s", entitySet)
	}

	id := body.Value[0]["ID"]
	if id == nil {
		return "", fmt.Errorf("entity in %s missing ID", entitySet)
	}
	return fmt.Sprintf("%v", id), nil
}

// collectEntityCollection follows every opaque continuation URL so assertions
// compare complete query results rather than only the first server-driven page.
func collectEntityCollection(ctx *framework.TestContext, path string, headers ...framework.Header) ([]map[string]interface{}, error) {
	resp, err := ctx.GET(path, headers...)
	if err != nil {
		return nil, err
	}
	return collectEntityPages(ctx, resp, headers...)
}

func collectEntityPages(ctx *framework.TestContext, resp *framework.HTTPResponse, headers ...framework.Header) ([]map[string]interface{}, error) {
	seenLinks := map[string]bool{}
	seenIDs := map[string]bool{}
	var all []map[string]interface{}
	for page := 0; page < 100; page++ {
		if err := ctx.AssertStatusCode(resp, 200); err != nil {
			return nil, err
		}
		items, err := ctx.ParseEntityCollection(resp)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			if id, ok := item["ID"]; ok {
				key := fmt.Sprint(id)
				if seenIDs[key] {
					return nil, fmt.Errorf("entity %s appears on multiple pages", key)
				}
				seenIDs[key] = true
			}
			all = append(all, item)
		}
		var envelope struct {
			NextLink string `json:"@odata.nextLink"`
		}
		if err := ctx.GetJSON(resp, &envelope); err != nil {
			return nil, err
		}
		if envelope.NextLink == "" {
			return all, nil
		}
		if seenLinks[envelope.NextLink] {
			return nil, fmt.Errorf("repeated @odata.nextLink %q", envelope.NextLink)
		}
		seenLinks[envelope.NextLink] = true
		resp, err = ctx.GETNextLink(envelope.NextLink, headers...)
		if err != nil {
			return nil, err
		}
	}
	return nil, fmt.Errorf("pagination exceeded 100 pages")
}
