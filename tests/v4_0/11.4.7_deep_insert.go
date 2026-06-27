package v4_0

import (
	"fmt"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// buildDeepInsertPayload builds a Product payload that contains inline related
// ProductDescription entities, exercising a real deep insert (creating an entity
// together with its related entities in a single request). The composite key of
// each description (ProductID + LanguageKey) is completed by the server from the
// parent it is created under, so only LanguageKey and the required Description
// need to be supplied.
func buildDeepInsertPayload(ctx *framework.TestContext, name string, price float64) (map[string]interface{}, error) {
	payload, err := buildProductPayload(ctx, name, price)
	if err != nil {
		return nil, err
	}
	payload["Descriptions"] = []map[string]interface{}{
		{"LanguageKey": "EN", "Description": name + " (EN)"},
		{"LanguageKey": "DE", "Description": name + " (DE)"},
	}
	return payload, nil
}

// DeepInsert creates the 11.4.7 Deep Insert test suite
func DeepInsert() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"11.4.7 Deep Insert",
		"Tests creating entities with related entities in a single request (deep insert).",
		"https://docs.oasis-open.org/odata/odata/v4.0/errata03/os/complete/part1-protocol/odata-v4.0-errata03-os-part1-protocol-complete.html#sec_CreateRelatedEntitiesWhenCreatinganE",
	)

	// Test 1: A deep insert (entity with inline related entities) returns 201 Created.
	suite.AddTest(
		"test_deep_insert_status",
		"Deep insert with nested entities returns 201 Created",
		func(ctx *framework.TestContext) error {
			payload, err := buildDeepInsertPayload(ctx, "Deep Insert Status Product", 50.00)
			if err != nil {
				return err
			}

			resp, err := ctx.POST("/Products", payload, framework.Header{Key: "Content-Type", Value: "application/json"})
			if err != nil {
				return err
			}

			return ctx.AssertStatusCode(resp, 201)
		},
	)

	// Test 2: The related entities supplied inline are actually created and
	// linked to the new parent. This is verified by reading the parent back with
	// $expand and confirming the nested entities are present.
	suite.AddTest(
		"test_deep_insert_creates_related_entities",
		"Deep insert creates and links the inline related entities",
		func(ctx *framework.TestContext) error {
			const name = "Deep Insert Related Product"
			payload, err := buildDeepInsertPayload(ctx, name, 75.00)
			if err != nil {
				return err
			}

			resp, err := ctx.POST("/Products", payload,
				framework.Header{Key: "Content-Type", Value: "application/json"},
				framework.Header{Key: "Prefer", Value: "return=representation"})
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 201); err != nil {
				return err
			}

			var created map[string]interface{}
			if err := ctx.GetJSON(resp, &created); err != nil {
				return fmt.Errorf("failed to parse create response: %w", err)
			}
			id, ok := created["ID"]
			if !ok || id == nil {
				return framework.NewError("created entity response is missing ID")
			}

			// Read the parent back with its related descriptions expanded.
			readResp, err := ctx.GET(fmt.Sprintf("/Products(%v)?$expand=Descriptions", id))
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(readResp, 200); err != nil {
				return err
			}

			var entity map[string]interface{}
			if err := ctx.GetJSON(readResp, &entity); err != nil {
				return fmt.Errorf("failed to parse expanded entity: %w", err)
			}
			rawDescriptions, ok := entity["Descriptions"]
			if !ok {
				return framework.NewError("expanded entity is missing the Descriptions navigation property")
			}
			descriptions, ok := rawDescriptions.([]interface{})
			if !ok {
				return framework.NewError("Descriptions is not a collection")
			}

			languages := map[string]bool{}
			for _, raw := range descriptions {
				desc, ok := raw.(map[string]interface{})
				if !ok {
					continue
				}
				if lang, ok := desc["LanguageKey"].(string); ok {
					languages[lang] = true
				}
			}
			for _, want := range []string{"EN", "DE"} {
				if !languages[want] {
					return framework.NewError(fmt.Sprintf("deep-inserted description %q was not created (got languages %v)", want, languages))
				}
			}

			return nil
		},
	)

	// Test 3: A deep insert is atomic. If a nested entity is invalid the whole
	// request must fail with 400 and the parent must NOT be created.
	suite.AddTest(
		"test_deep_insert_invalid_nested_is_atomic",
		"Deep insert with an invalid nested entity fails atomically (400, parent not created)",
		func(ctx *framework.TestContext) error {
			const name = "Deep Insert Atomicity Probe"
			payload, err := buildProductPayload(ctx, name, 60.00)
			if err != nil {
				return err
			}
			// The parent is valid, but the nested description is invalid: the
			// required string Description is given a structured value.
			payload["Descriptions"] = []map[string]interface{}{
				{"LanguageKey": "EN", "Description": map[string]interface{}{"unexpected": "object"}},
			}

			resp, err := ctx.POST("/Products", payload, framework.Header{Key: "Content-Type", Value: "application/json"})
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(resp, 400); err != nil {
				return err
			}

			// Atomicity: the parent must not have been persisted.
			check, err := ctx.GET(fmt.Sprintf("/Products?$filter=Name eq '%s'&$top=1", name))
			if err != nil {
				return err
			}
			if err := ctx.AssertStatusCode(check, 200); err != nil {
				return err
			}
			items, err := ctx.ParseEntityCollection(check)
			if err != nil {
				return err
			}
			if len(items) != 0 {
				return framework.NewError("deep insert was not atomic: parent entity was created despite an invalid nested entity")
			}

			return nil
		},
	)

	// Test 4: Deep insert returns a Location header for the created parent.
	suite.AddTest(
		"test_deep_insert_location_header",
		"Deep insert returns Location header",
		func(ctx *framework.TestContext) error {
			payload, err := buildDeepInsertPayload(ctx, "Deep Insert Location Product", 80.00)
			if err != nil {
				return err
			}

			resp, err := ctx.POST("/Products", payload, framework.Header{Key: "Content-Type", Value: "application/json"})
			if err != nil {
				return err
			}

			if err := ctx.AssertStatusCode(resp, 201); err != nil {
				return err
			}

			if resp.Headers.Get("Location") == "" {
				return framework.NewError("Location header not found")
			}

			return nil
		},
	)

	return suite
}
