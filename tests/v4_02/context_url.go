package v4_02

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/nlstn/odata-compliance-suite/framework"
)

func ContextURL() *framework.TestSuite {
	suite := framework.NewTestSuite("10.2 OData 4.02 context URL", "Checks canonical parentheses keys and unescaped context fragments.", protocolSpec+"#sec_ContextURL")
	suite.AddTest("test_canonical_property_context", "A property context fragment formats the composite key canonically", func(ctx *framework.TestContext) error {
		items, err := readCollection(ctx, "/ProductDescriptions?$filter=LanguageKey%20eq%20'EN'&$top=1", max402)
		if err != nil {
			return err
		}
		if len(items) != 1 {
			return fmt.Errorf("expected one seeded EN ProductDescription")
		}
		id, ok := items[0]["ProductID"].(string)
		if !ok || id == "" {
			return fmt.Errorf("seeded ProductDescription lacks ProductID")
		}
		path := fmt.Sprintf("/ProductDescriptions(LanguageKey=%%27%%45N%%27,ProductID=%s)/Description", id)
		resp, err := ctx.GET(path, max402)
		if err != nil {
			return err
		}
		if err := assertResponse(ctx, resp, http.StatusOK, "4.02"); err != nil {
			return err
		}
		var body map[string]interface{}
		if err := ctx.GetJSON(resp, &body); err != nil {
			return err
		}
		context, _ := body["@context"].(string)
		if context == "" {
			context, _ = body["@odata.context"].(string)
		}
		_, fragment, found := strings.Cut(context, "#")
		want := fmt.Sprintf("ProductDescriptions(ProductID=%s,LanguageKey='EN')/Description", id)
		if !found || fragment != want {
			return fmt.Errorf("property context fragment = %q, want %q", fragment, want)
		}
		if body["value"] != items[0]["Description"] {
			return fmt.Errorf("property response does not contain the seeded description")
		}
		return nil
	})
	return suite
}
