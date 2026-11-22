package v4_02

import (
	"fmt"
	"net/http"
	"net/url"
	"reflect"

	"github.com/nlstn/odata-compliance-suite/framework"
)

func SelectKeys() *framework.TestSuite {
	suite := framework.NewTestSuite("5.1.4 OData 4.02 select keys", "Checks $key projection with simple, composite, and navigation keys.", urlSpec+"#sec_SystemQueryOption_select")
	suite.AddTest("test_select_key", "$select=$key,Name projects a single key alongside the requested property", func(ctx *framework.TestContext) error {
		path := "/Products?$filter=Name%20eq%20'Laptop'&$select=$key,Name"
		items, err := readCollection(ctx, path, max402)
		if err != nil {
			return err
		}
		if err := assertSeededLaptop(items); err != nil {
			return err
		}
		if _, present := items[0]["Price"]; present {
			return fmt.Errorf("unexpected Price in a key and Name projection")
		}
		return nil
	})
	suite.AddTest("test_select_composite_key", "$select=$key includes every component of a composite key", func(ctx *framework.TestContext) error {
		items, err := readCollection(ctx, "/ProductDescriptions?$select=$key&$top=1", max402)
		if err != nil {
			return err
		}
		if len(items) != 1 || items[0]["ProductID"] == nil || items[0]["LanguageKey"] == nil {
			return fmt.Errorf("expected both components of the ProductDescription key, got %v", items)
		}
		if _, present := items[0]["Description"]; present {
			return fmt.Errorf("unexpected Description in a key-only projection")
		}
		return nil
	})
	suite.AddTest("test_nested_select_key", "$key resolves against the expanded entity type", func(ctx *framework.TestContext) error {
		items, err := readCollection(ctx, "/Products?$filter=Name%20eq%20'Laptop'&$expand=Descriptions($select=$key)", max402)
		if err != nil {
			return err
		}
		if err := assertSeededLaptop(items); err != nil {
			return err
		}
		children, ok := items[0]["Descriptions"].([]interface{})
		if !ok || len(children) == 0 {
			return fmt.Errorf("Laptop must have expanded Descriptions")
		}
		for _, child := range children {
			entity, ok := child.(map[string]interface{})
			if !ok || entity["ProductID"] != items[0]["ID"] || entity["LanguageKey"] == nil {
				return fmt.Errorf("expanded Description has an incomplete or incorrect key: %v", child)
			}
			if _, present := entity["Description"]; present {
				return fmt.Errorf("unexpected Description in a nested key-only projection")
			}
		}
		return nil
	})
	return suite
}

func Expand() *framework.TestSuite {
	suite := framework.NewTestSuite("5.1.3 OData 4.02 expand options", "Checks empty expansions and rejects duplicate options in an expand item.", urlSpec+"#sec_SystemQueryOption_expand")
	suite.AddTest("test_empty_expand", "An empty $expand is valid and contains no expanded navigation properties", func(ctx *framework.TestContext) error {
		path := "/Products?$filter=Name%20eq%20'Laptop'"
		baseline, err := readCollection(ctx, path, max402)
		if err != nil {
			return err
		}
		items, err := readCollection(ctx, path+"&$expand=", max402)
		if err != nil {
			return err
		}
		if err := assertSeededLaptop(items); err != nil {
			return err
		}
		if !reflect.DeepEqual(baseline, items) {
			return fmt.Errorf("empty $expand changed the entity representation")
		}
		if _, present := items[0]["Descriptions"]; present {
			return fmt.Errorf("empty $expand expanded Descriptions")
		}
		return nil
	})
	suite.AddTest("test_nested_empty_expand", "An empty nested $expand leaves the containing expansion intact", func(ctx *framework.TestContext) error {
		items, err := readCollection(ctx, "/Products?$filter=Name%20eq%20'Laptop'&$expand=Descriptions($expand=)", max402)
		if err != nil {
			return err
		}
		if err := assertSeededLaptop(items); err != nil {
			return err
		}
		children, ok := items[0]["Descriptions"].([]interface{})
		if !ok || len(children) == 0 {
			return fmt.Errorf("empty nested expand removed the containing expansion")
		}
		for _, child := range children {
			entity, ok := child.(map[string]interface{})
			if !ok {
				return fmt.Errorf("expanded Description is not an object")
			}
			if _, present := entity["Product"]; present {
				return fmt.Errorf("empty nested expand expanded Product")
			}
		}
		return nil
	})
	suite.AddTest("test_duplicate_expand_option", "A nested system query option can occur only once in an expand item", func(ctx *framework.TestContext) error {
		for _, expand := range []string{
			"Descriptions($top=1;$top=2)",
			"Descriptions($top=1;ToP=2)",
		} {
			resp, err := ctx.GET("/Products?$expand="+url.QueryEscape(expand), max402)
			if err != nil {
				return err
			}
			if err := ctx.AssertODataError(resp, http.StatusBadRequest, ""); err != nil {
				return fmt.Errorf("$expand=%s: %w", expand, err)
			}
		}
		return nil
	})
	return suite
}

func MatchesPattern() *framework.TestSuite {
	suite := framework.NewTestSuite("5.1.1.7.1 OData 4.02 matchespattern flags", "Checks case-insensitive matches and null results for invalid flag values.", urlSpec+"#sec_matchespattern")
	suite.AddTest("test_matchespattern_case_insensitive", "The i flag changes the matchespattern result", func(ctx *framework.TestContext) error {
		for _, tc := range []struct {
			flags string
			want  int
		}{{"''", 0}, {"'i'", 1}} {
			filter := "matchesPattern(Name,'^laptop$'," + tc.flags + ")"
			items, err := readCollection(ctx, "/Products?$filter="+url.QueryEscape(filter), max402)
			if err != nil {
				return err
			}
			if len(items) != tc.want {
				return fmt.Errorf("%s returned %d rows, want %d", filter, len(items), tc.want)
			}
			if tc.want == 1 {
				if err := assertSeededLaptop(items); err != nil {
					return err
				}
			}
		}
		return nil
	})
	suite.AddTest("test_matchespattern_invalid_flags", "Invalid or repeated flags produce null, even under not", func(ctx *framework.TestContext) error {
		for _, flags := range []string{"z", "ii"} {
			filter := "not matchesPattern(Name,'^laptop$','" + flags + "')"
			items, err := readCollection(ctx, "/Products?$filter="+url.QueryEscape(filter), max402)
			if err != nil {
				return err
			}
			if len(items) != 0 {
				return fmt.Errorf("negated matchespattern with invalid flags %q returned %d items", flags, len(items))
			}
		}
		return nil
	})
	return suite
}
