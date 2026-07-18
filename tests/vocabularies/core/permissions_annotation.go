package core

import (
	"fmt"
	"strings"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// PermissionsAnnotation creates tests for the Core.Permissions annotation
// Tests that entity sets annotated with a read-only Core.Permissions value
// reject write operations.
func PermissionsAnnotation() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"Core.Permissions Annotation",
		"Validates that entity sets annotated with Org.OData.Core.V1.Permissions=Read reject write operations.",
		"https://github.com/oasis-tcs/odata-vocabularies/blob/master/vocabularies/Org.OData.Core.V1.md#Permissions",
	)

	suite.AddTest(
		"metadata_includes_permissions_annotation",
		"Metadata document includes Core.Permissions annotations where defined",
		func(ctx *framework.TestContext) error {
			resp, err := ctx.GET("/$metadata", framework.Header{Key: "Accept", Value: "application/xml"})
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 200); err != nil {
				return err
			}

			hits, err := findAnnotationsByTerm(resp.Body, "Core.Permissions")
			if err != nil {
				return err
			}
			if len(hits) == 0 {
				return ctx.Skip("Core.Permissions is an optional annotation not used by this model")
			}
			return nil
		},
	)

	suite.AddTest(
		"read_only_entity_set_rejects_post",
		"POST to an entity set annotated Core.Permissions=Read (or Core.Permissions=None) returns 4xx",
		func(ctx *framework.TestContext) error {
			metadataResp, err := ctx.GET("/$metadata", framework.Header{Key: "Accept", Value: "application/xml"})
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(metadataResp, 200); err != nil {
				return err
			}

			hits, err := findAnnotationsByTerm(metadataResp.Body, "Core.Permissions")
			if err != nil {
				return err
			}

			var readOnlySets []string
			for _, hit := range hits {
				if hit.EnumMember == nil {
					continue
				}
				member := *hit.EnumMember
				if !strings.HasSuffix(member, "/Read") && !strings.HasSuffix(member, "/None") {
					continue // Write/Insert/Update/Delete/ReadWrite grant some write access
				}
				// Target for an entity-set-level Permissions annotation looks like
				// "Namespace.Container/EntitySetName".
				idx := strings.LastIndex(hit.Target, "/")
				if idx == -1 || !strings.Contains(hit.Target[:idx], ".Container") {
					continue // property- or type-level target; not an entity-set restriction
				}
				readOnlySets = append(readOnlySets, hit.Target[idx+1:])
			}

			if len(readOnlySets) == 0 {
				return ctx.Skip("no entity set has a read-only Core.Permissions annotation in this model")
			}

			for _, setName := range readOnlySets {
				// Send an otherwise-valid payload so a rejection can only be
				// attributed to the read-only Permissions annotation, not to a
				// missing required field.
				payload, err := buildValidCreatePayload(metadataResp.Body, setName)
				if err != nil {
					return err
				}
				resp, err := ctx.POST(fmt.Sprintf("/%s", setName), payload)
				if err != nil {
					return err
				}
				if resp.StatusCode < 400 || resp.StatusCode >= 500 {
					return fmt.Errorf("expected 4xx for POST to read-only entity set %s, got %d: %s", setName, resp.StatusCode, string(resp.Body))
				}
				if err := assertODataError(resp); err != nil {
					return fmt.Errorf("read-only entity set %s: response is not a well-formed OData error: %w", setName, err)
				}
			}
			return nil
		},
	)

	return suite
}
