package v4_02

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/nlstn/odata-compliance-suite/framework"
)

func JSONBatch() *framework.TestSuite {
	suite := framework.NewTestSuite("19 OData 4.02 JSON batch", "Checks ordered requests, implicit group dependencies, and error continuation.", jsonSpec+"#sec_BatchRequestsandResponses")
	suite.AddTest("test_batch_streaming_order", "4.02 streaming batch requests require id first and body last", func(ctx *framework.TestContext) error {
		for _, body := range []string{
			`{"requests":[{"method":"GET","id":"read","url":"Products?$top=1"}]}`,
			`{"requests":[{"id":"write","method":"POST","body":{"Name":"Invalid order"},"url":"ReadOnlyItems"}]}`,
		} {
			resp, err := ctx.POSTRaw("/$batch", []byte(body), "application/json;streaming=true", payload402, max402)
			if err != nil {
				return err
			}
			if err := ctx.AssertODataError(resp, http.StatusBadRequest, ""); err != nil {
				return err
			}
		}
		for _, version := range []string{"4.01", "4.02"} {
			body := `{"requests":[{"id":"read","method":"GET","url":"Products?$top=1"}]}`
			if version == "4.01" {
				body = `{"requests":[{"method":"GET","id":"read","url":"Products?$top=1"}]}`
			}
			resp, err := ctx.POSTRaw("/$batch", []byte(body), "application/json;streaming=true",
				framework.Header{Key: "OData-Version", Value: version}, framework.Header{Key: "OData-MaxVersion", Value: version})
			if err != nil {
				return err
			}
			if err := assertResponse(ctx, resp, http.StatusOK, version); err != nil {
				return err
			}
			parts, err := batchStatuses(resp)
			if err != nil {
				return err
			}
			if parts["read"] != http.StatusOK {
				return fmt.Errorf("ordered batch read status = %d", parts["read"])
			}
		}
		return nil
	})
	suite.AddTest("test_batch_atomic_group_dependency", "A dependency on a member of a failed atomicity group yields 424", func(ctx *framework.TestContext) error {
		// Read-only requests keep the fixture unchanged even on a service that
		// accidentally executes the dependent operation.
		body := `{"requests":[
			{"id":"first","method":"GET","url":"Products?$top=1","atomicityGroup":"group"},
			{"id":"fail","method":"GET","url":"Products(00000000-0000-0000-0000-000000000000)","atomicityGroup":"group"},
			{"id":"dependent","method":"GET","url":"Products?$top=1","dependsOn":["first"]},
			{"id":"unrelated","method":"GET","url":"Products?$top=1"}
		]}`
		resp, err := ctx.POSTRaw("/$batch", []byte(body), "application/json", payload402, max402,
			framework.Header{Key: "Prefer", Value: "continue-on-error"})
		if err != nil {
			return err
		}
		if err := assertResponse(ctx, resp, http.StatusOK, "4.02"); err != nil {
			return err
		}
		parts, err := batchStatuses(resp)
		if err != nil {
			return err
		}
		if parts["fail"] != http.StatusNotFound || parts["dependent"] != http.StatusFailedDependency || parts["unrelated"] != http.StatusOK {
			return fmt.Errorf("unexpected batch response statuses: %v", parts)
		}
		for _, part := range strings.Split(strings.Join(resp.Headers.Values("Preference-Applied"), ","), ",") {
			if strings.EqualFold(strings.TrimSpace(part), "continue-on-error=true") {
				return nil
			}
		}
		return fmt.Errorf("batch continued after an error without Preference-Applied: continue-on-error=true")
	})
	suite.AddTest("test_batch_committed_group_dependency", "A dependent request succeeds when its prerequisite group commits", func(ctx *framework.TestContext) error {
		body := `{"requests":[
			{"id":"first","method":"GET","url":"Products?$top=1","atomicityGroup":"group"},
			{"id":"second","method":"GET","url":"Categories?$top=1","atomicityGroup":"group"},
			{"id":"dependent","method":"GET","url":"Products?$top=1","dependsOn":["first"]}
		]}`
		resp, err := ctx.POSTRaw("/$batch", []byte(body), "application/json", payload402, max402)
		if err != nil {
			return err
		}
		if err := assertResponse(ctx, resp, http.StatusOK, "4.02"); err != nil {
			return err
		}
		statuses, err := batchStatuses(resp)
		if err != nil {
			return err
		}
		for _, id := range []string{"first", "second", "dependent"} {
			if statuses[id] != http.StatusOK {
				return fmt.Errorf("expected successful batch part %q, got statuses %v", id, statuses)
			}
		}
		return nil
	})

	return suite
}

func batchStatuses(resp *framework.HTTPResponse) (map[string]int, error) {
	var batch struct {
		Responses []struct {
			ID     string `json:"id"`
			Status int    `json:"status"`
		} `json:"responses"`
	}
	if err := json.Unmarshal(resp.Body, &batch); err != nil {
		return nil, err
	}
	statuses := make(map[string]int, len(batch.Responses))
	for _, part := range batch.Responses {
		if _, duplicate := statuses[part.ID]; part.ID == "" || duplicate || part.Status == 0 {
			return nil, fmt.Errorf("invalid or duplicate batch response: %+v", part)
		}
		statuses[part.ID] = part.Status
	}
	return statuses, nil
}
