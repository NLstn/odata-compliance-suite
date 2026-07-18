package v4_0

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// TypeDefinitions creates the 5.4 Type Definitions test suite
func TypeDefinitions() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"5.4 Type Definitions",
		"Validates custom type definitions in metadata document and their proper usage.",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part3-csdl/odata-v4.0-errata03-os-part3-csdl-complete.html#sec_TypeDefinition",
	)

	suite.AddTest(
		"test_typedef_in_metadata",
		"Metadata contains valid schema",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			body := string(resp.Body)
			if !strings.Contains(body, "<Schema") && !strings.Contains(body, `"$Version"`) {
				return framework.NewError("Metadata does not contain valid schema")
			}

			return nil
		},
	)

	suite.AddTest(
		"test_typedef_underlying_type",
		"Each TypeDefinition declares a valid UnderlyingType, and at least one structural property actually uses one",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata")
			if err != nil {
				return err
			}

			body := string(resp.Body)
			typeDefPattern := regexp.MustCompile(`<TypeDefinition\s[^>]*/?>`)
			typeDefs := typeDefPattern.FindAllString(body, -1)
			if len(typeDefs) == 0 {
				return ctx.Skip("no TypeDefinition declared in metadata")
			}

			nameAndUnderlyingPattern := regexp.MustCompile(`Name="([^"]+)"[^>]*UnderlyingType="([^"]+)"`)
			typeDefNames := map[string]bool{}
			for _, td := range typeDefs {
				m := nameAndUnderlyingPattern.FindStringSubmatch(td)
				if m == nil {
					return fmt.Errorf("TypeDefinition is missing a valid Name/UnderlyingType attribute pair: %s", td)
				}
				typeDefNames[m[1]] = true
			}

			ns, err := schemaNamespace(ctx)
			if err != nil {
				return err
			}
			if ns == "" {
				return ctx.Skip("could not determine the schema namespace")
			}

			used := false
			for name := range typeDefNames {
				if strings.Contains(body, `Type="`+ns+"."+name+`"`) {
					used = true
					break
				}
			}
			if !used {
				return fmt.Errorf("metadata declares TypeDefinition(s) but no structural property references one by its qualified type name")
			}
			return nil
		},
	)

	return suite
}
