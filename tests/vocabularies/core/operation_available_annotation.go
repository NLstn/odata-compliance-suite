package core

import (
	"fmt"
	"strings"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// OperationAvailableAnnotation creates tests for the Core.OperationAvailable annotation
// Tests that an unbound action/function import statically annotated
// Core.OperationAvailable=false is not invocable.
func OperationAvailableAnnotation() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"Core.OperationAvailable Annotation",
		"Validates that action/function imports statically annotated Org.OData.Core.V1.OperationAvailable=false are rejected when invoked.",
		"https://github.com/oasis-tcs/odata-vocabularies/blob/master/vocabularies/Org.OData.Core.V1.md#OperationAvailable",
	)

	suite.AddTest(
		"metadata_includes_operation_available_annotation",
		"Metadata document includes Core.OperationAvailable annotations where defined",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata", framework.Header{Key: "Accept", Value: "application/xml"})
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			hits, err := findAnnotationsByTerm(resp.Body, "Core.OperationAvailable")
			if err != nil {
				return err
			}
			if len(hits) == 0 {
				return ctx.Skip("Core.OperationAvailable is an optional annotation not used by this model")
			}
			return nil
		},
	)

	suite.AddTest(
		"unavailable_operation_import_rejects_invocation",
		"Invoking an action/function import statically annotated OperationAvailable=false returns 4xx/405",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata", framework.Header{Key: "Accept", Value: "application/xml"})
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			hits, err := findAnnotationsByTerm(resp.Body, "Core.OperationAvailable")
			if err != nil {
				return err
			}

			var unavailableImports []string
			for _, hit := range hits {
				// Only the static Bool form can be checked without evaluating a
				// dynamic Path expression against live entity data.
				if hit.Bool == nil || *hit.Bool {
					continue
				}
				// Target must directly name a container-level Action/FunctionImport
				// ("Namespace.Container/ImportName") to be invocable without also
				// knowing its binding parameter and HTTP method.
				idx := strings.LastIndex(hit.Target, "/")
				if idx == -1 || !strings.Contains(hit.Target[:idx], ".Container") {
					continue
				}
				unavailableImports = append(unavailableImports, hit.Target[idx+1:])
			}

			if len(unavailableImports) == 0 {
				return ctx.Skip("no action/function import has a static OperationAvailable=false annotation in this model")
			}

			for _, importName := range unavailableImports {
				resp, err := ctx.POST(fmt.Sprintf("/%s", importName), map[string]interface{}{})
				if err != nil {
					return err
				}
				if resp.StatusCode < 400 || resp.StatusCode >= 500 {
					return fmt.Errorf("expected 4xx for invoking unavailable operation %s, got %d: %s", importName, resp.StatusCode, string(resp.Body))
				}
			}
			return nil
		},
	)

	return suite
}
