package v4_0

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/nlstn/odata-compliance-suite/framework"
)

func ApplyNested() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.2.5.4.6 $apply nested transformations",
		"Tests strict-spec addnested and nest output structure and nested transformation semantics.",
		"https://docs.oasis-open.org/odata/odata-data-aggregation-ext/v4.0/cs03/odata-data-aggregation-ext-v4.0-cs03.html#Transformationaddnested",
	)
	suite.AddTest("test_apply_addnested_filtered_navigation", "addnested adds a filtered dynamic collection property per input entity", func(ctx *framework.TestContext) error {
		expr := "addnested(Descriptions,filter(LanguageKey eq 'EN') as EnglishDescriptions)"
		resp, err := ctx.GET("/Products?$apply=" + url.QueryEscape(expr))
		if err != nil {
			return err
		}
		if err := ctx.AssertStatusCode(resp, 200); err != nil {
			return err
		}
		var body struct {
			Value []map[string]interface{} `json:"value"`
		}
		if err := json.Unmarshal(resp.Body, &body); err != nil {
			return err
		}
		if len(body.Value) != 7 {
			return fmt.Errorf("expected seven products, got %d", len(body.Value))
		}
		for _, product := range body.Value {
			if _, ok := product["ID"]; !ok {
				return fmt.Errorf("addnested result lost input entity: %#v", product)
			}
			nested, ok := product["EnglishDescriptions"].([]interface{})
			if !ok {
				return fmt.Errorf("EnglishDescriptions must be an array, got %T", product["EnglishDescriptions"])
			}
			for _, item := range nested {
				row, ok := item.(map[string]interface{})
				if !ok || row["LanguageKey"] != "EN" {
					return fmt.Errorf("nested result contains non-EN description: %#v", item)
				}
			}
		}
		return nil
	})
	suite.AddTest("test_apply_nest_groupby", "nest returns one row containing the transformed input collection", func(ctx *framework.TestContext) error {
		expr := "nest(groupby((CategoryID)) as Categories)"
		resp, err := ctx.GET("/Products?$apply=" + url.QueryEscape(expr))
		if err != nil {
			return err
		}
		if err := ctx.AssertStatusCode(resp, 200); err != nil {
			return err
		}
		var body struct {
			Value []map[string]interface{} `json:"value"`
		}
		if err := json.Unmarshal(resp.Body, &body); err != nil {
			return err
		}
		if len(body.Value) != 1 {
			return fmt.Errorf("nest must return one row, got %d", len(body.Value))
		}
		categories, ok := body.Value[0]["Categories"].([]interface{})
		if !ok || len(categories) == 0 {
			return fmt.Errorf("Categories must be a non-empty array, got %#v", body.Value[0]["Categories"])
		}
		return nil
	})
	return suite
}
