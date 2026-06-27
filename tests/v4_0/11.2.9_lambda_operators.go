package v4_0

import (
	"strings"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// LambdaOperators creates a test suite for lambda operators.
//
// Each test verifies the lambda's actual semantics: the filtered set is compared
// against an oracle computed in Go from the products' expanded Descriptions (see
// assertProductLambdaFilter), not merely checked for HTTP 200.
func LambdaOperators() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.2.9 Lambda Operators (any, all)",
		"Tests lambda operators for collection navigation and filtering",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part2-url-conventions/odata-v4.0-errata03-os-part2-url-conventions-complete.html#sec_LambdaOperators",
	)
	RegisterLambdaOperatorsTests(suite)
	return suite
}

// hasDescription reports whether any of a product's descriptions satisfies pred.
func hasDescription(p map[string]interface{}, pred func(d map[string]interface{}) bool) bool {
	for _, d := range productDescriptions(p) {
		if pred(d) {
			return true
		}
	}
	return false
}

// RegisterLambdaOperatorsTests registers tests for lambda operators (any, all)
func RegisterLambdaOperatorsTests(suite *framework.TestSuite) {
	// any(): at least one related description has LanguageKey 'EN'.
	suite.AddTest(
		"Lambda any operator with collection navigation",
		"any() returns entities with at least one matching related entity",
		func(ctx *framework.TestContext) error {
			return assertProductLambdaFilter(ctx, "Descriptions/any(d: d/LanguageKey eq 'EN')",
				func(p map[string]interface{}) bool {
					return hasDescription(p, func(d map[string]interface{}) bool {
						return productString(d, "LanguageKey") == "EN"
					})
				})
		})

	// all(): every related description satisfies the predicate; vacuously true for
	// products with no descriptions.
	suite.AddTest(
		"Lambda all operator with collection navigation",
		"all() returns entities whose related entities all match (incl. empty collections)",
		func(ctx *framework.TestContext) error {
			return assertProductLambdaFilter(ctx, "Descriptions/all(d: d/LanguageKey ne 'XX')",
				func(p map[string]interface{}) bool {
					for _, d := range productDescriptions(p) {
						if productString(d, "LanguageKey") == "XX" {
							return false
						}
					}
					return true
				})
		})

	// any() with a compound boolean predicate inside the lambda.
	suite.AddTest(
		"Lambda any with complex condition",
		"any() with a compound boolean predicate inside the lambda",
		func(ctx *framework.TestContext) error {
			return assertProductLambdaFilter(ctx, "Descriptions/any(d: d/LanguageKey eq 'EN' and d/CustomName eq 'Promo')",
				func(p map[string]interface{}) bool {
					return hasDescription(p, func(d map[string]interface{}) bool {
						return productString(d, "LanguageKey") == "EN" && productString(d, "CustomName") == "Promo"
					})
				})
		})

	// any() with a string function applied to a related property.
	suite.AddTest(
		"Lambda any with property comparison",
		"any() with a string function applied to a related property",
		func(ctx *framework.TestContext) error {
			return assertProductLambdaFilter(ctx, "Descriptions/any(d: contains(d/Description,'Ergonomic'))",
				func(p map[string]interface{}) bool {
					return hasDescription(p, func(d map[string]interface{}) bool {
						return strings.Contains(productString(d, "Description"), "Ergonomic")
					})
				})
		})

	// Two any() predicates combined at the top level.
	suite.AddTest(
		"Nested lambda operators",
		"Two any() predicates combined with and at the top level",
		func(ctx *framework.TestContext) error {
			return assertProductLambdaFilter(ctx, "Descriptions/any(d: d/LanguageKey eq 'EN') and Descriptions/any(d: d/LanguageKey eq 'DE')",
				func(p map[string]interface{}) bool {
					hasEN := hasDescription(p, func(d map[string]interface{}) bool { return productString(d, "LanguageKey") == "EN" })
					hasDE := hasDescription(p, func(d map[string]interface{}) bool { return productString(d, "LanguageKey") == "DE" })
					return hasEN && hasDE
				})
		})

	// any() referencing a nullable related property (CustomName).
	suite.AddTest(
		"Lambda any with custom column mapping",
		"any() referencing a nullable related property",
		func(ctx *framework.TestContext) error {
			return assertProductLambdaFilter(ctx, "Descriptions/any(d: d/CustomName eq 'Promo')",
				func(p map[string]interface{}) bool {
					return hasDescription(p, func(d map[string]interface{}) bool {
						return productString(d, "CustomName") == "Promo"
					})
				})
		})
}
