package v4_0

import (
	"fmt"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// collectEntityCollection follows every server-provided continuation link.
// It keeps the link opaque and detects loops and duplicate entities.
func collectEntityCollection(ctx *framework.TestContext, path string, headers ...framework.Header) ([]map[string]interface{}, error) {
	resp, err := ctx.GET(path, headers...)
	if err != nil {
		return nil, err
	}
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
				key := fmt.Sprintf("%v", id)
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
