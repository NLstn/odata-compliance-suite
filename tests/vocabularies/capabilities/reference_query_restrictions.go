package capabilities

import (
	"fmt"
	"net/url"

	"github.com/nlstn/odata-compliance-suite/framework"
)

const restrictedFixtureSet = "ReadOnlyItems"

// ReferenceQueryRestrictions verifies that the reference model contains a
// deliberately restricted entity set. This keeps the individual vocabulary
// suites from silently passing by skipping every negative-path check.
func ReferenceQueryRestrictions() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"Capabilities query-restriction reference fixture",
		"Validates the required ReadOnlyItems capability annotations and their observable query behavior.",
		"https://oasis-tcs.github.io/odata-vocabularies/vocabularies/Org.OData.Capabilities.V1.html",
	)

	suite.AddTest(
		"metadata_declares_required_query_restrictions",
		"ReadOnlyItems declares Filterable, Sortable, Expandable, Countable, Searchable, and SelectSupport as false",
		func(ctx *framework.TestContext) error {
			metadataXML, err := fetchMetadata(ctx)
			if err != nil {
				return err
			}
			metadataInfo, err := parseCapabilitiesMetadata(metadataXML)
			if err != nil {
				return err
			}

			required := []struct {
				name string
				sets []entitySetInfo
			}{
				{"FilterRestrictions.Filterable", metadataInfo.filterRestricted},
				{"SortRestrictions.Sortable", metadataInfo.sortRestricted},
				{"ExpandRestrictions.Expandable", metadataInfo.expandRestricted},
				{"CountRestrictions.Countable", metadataInfo.countRestricted},
				{"SearchRestrictions.Searchable", metadataInfo.searchRestricted},
				{"SelectSupport.Supported", metadataInfo.selectRestricted},
			}
			for _, requirement := range required {
				if !containsEntitySet(requirement.sets, restrictedFixtureSet) {
					return fmt.Errorf("%s must declare %s=false in the reference metadata", restrictedFixtureSet, requirement.name)
				}
			}
			return nil
		},
	)

	for _, tc := range []struct {
		name  string
		query string
	}{
		{"filter_restriction_is_enforced", "$filter=" + url.QueryEscape("Name ne null")},
		{"sort_restriction_is_enforced", "$orderby=" + url.QueryEscape("Name asc")},
		{"expand_restriction_is_enforced", "$expand=*"},
		{"count_restriction_is_enforced", "$count=true"},
		{"search_restriction_is_enforced", "$search=" + url.QueryEscape("item")},
		{"select_restriction_is_enforced", "$select=" + url.QueryEscape("ID,Name")},
	} {
		tc := tc
		suite.AddTest(
			tc.name,
			"A query prohibited by the advertised ReadOnlyItems capabilities returns an OData 400 response",
			func(ctx *framework.TestContext) error {
				resp, err := ctx.GET("/" + restrictedFixtureSet + "?" + tc.query)
				if err != nil {
					return err
				}
				return ctx.AssertODataError(resp, 400, "")
			},
		)
	}

	return suite
}

func containsEntitySet(sets []entitySetInfo, name string) bool {
	for _, set := range sets {
		if set.name == name {
			return true
		}
	}
	return false
}
