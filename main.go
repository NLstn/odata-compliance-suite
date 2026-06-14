package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/nlstn/odata-compliance-suite/framework"
	v4_0 "github.com/nlstn/odata-compliance-suite/tests/v4_0"
	v4_01 "github.com/nlstn/odata-compliance-suite/tests/v4_01"
	"github.com/nlstn/odata-compliance-suite/tests/vocabularies/capabilities"
	"github.com/nlstn/odata-compliance-suite/tests/vocabularies/core"
)

// buildVersion is stamped at build time via -ldflags "-X main.buildVersion=...".
// It defaults to "dev" for `go run` / `go build` without ldflags.
var buildVersion = "dev"

var (
	serverURL = flag.String("server", "http://localhost:9090", "URL of the OData service under test")
	version   = flag.String("version", "all", "OData version to test (4.0, 4.01, vocabularies, or all)")
	pattern   = flag.String("pattern", "", "Run only suites whose name matches this substring")
	debug     = flag.Bool("debug", false, "Enable debug mode with full HTTP request/response details")
	verbose   = flag.Bool("verbose", false, "Enable verbose mode to show all individual test results")
	timeout   = flag.Int("timeout", 30, "Seconds to wait for the server to become reachable before giving up")
	strict    = flag.Bool("strict", false, "Treat capability-skipped tests as failures (disables capability-aware skipping)")
)

type TestSuiteInfo struct {
	Name    string
	Version string
	Suite   func() *framework.TestSuite
}

func main() {
	flag.Parse()

	fmt.Println()
	fmt.Println("╔════════════════════════════════════════════════════════╗")
	fmt.Println("║     OData v4 Compliance Test Suite                     ║")
	fmt.Println("╚════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Printf("Suite Ver:  %s\n", buildVersion)
	fmt.Printf("Server URL: %s\n", *serverURL)
	fmt.Printf("Version:    %s\n", *version)
	if *debug {
		fmt.Println("Debug Mode: ENABLED")
	}
	if *verbose {
		fmt.Println("Verbose Mode: ENABLED")
	}
	fmt.Println()

	// The suite runs entirely against an externally-provided OData service.
	// Wait for it to become reachable before running any tests.
	if !waitForServer(*serverURL, *timeout) {
		fmt.Fprintf(os.Stderr, "Error: cannot reach OData service at %s after %ds\n", *serverURL, *timeout)
		fmt.Fprintln(os.Stderr, "Start your service and point -server at its root URL.")
		fmt.Fprintln(os.Stderr, "The service must expose the reference data model documented in CONTRACT.md.")
		os.Exit(1)
	}

	// Fetch and parse the service's $metadata to build a capability profile used for
	// intelligent test skipping. Failure is non-fatal: we warn and run all tests.
	var capProfile *framework.CapabilityProfile
	if !*strict {
		if profile, err := fetchCapabilityProfile(*serverURL); err != nil {
			fmt.Printf("⚠ WARNING: Could not parse capability profile from $metadata: %v\n", err)
			fmt.Println("  Capability-aware skipping is disabled; all tests will run.")
			fmt.Println()
		} else {
			capProfile = profile
		}
	}

	// Gather test suites
	testSuites := []TestSuiteInfo{}

	// Register v4.0 tests
	if *version == "all" || *version == "4.0" {
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "1.1_introduction",
			Version: "4.0",
			Suite:   v4_0.Introduction,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "2.1_conformance",
			Version: "4.0",
			Suite:   v4_0.Conformance,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "3.1_edmx_element",
			Version: "4.0",
			Suite:   v4_0.EDMXElement,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "3.2_dataservices_element",
			Version: "4.0",
			Suite:   v4_0.DataServicesElement,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "3.3_reference_element",
			Version: "4.0",
			Suite:   v4_0.ReferenceElement,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "3.4_include_element",
			Version: "4.0",
			Suite:   v4_0.IncludeElement,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "3.5_includeannotations_element",
			Version: "4.0",
			Suite:   v4_0.IncludeAnnotationsElement,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "4.1_nominal_types",
			Version: "4.0",
			Suite:   v4_0.NominalTypes,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "4.2_structured_types",
			Version: "4.0",
			Suite:   v4_0.StructuredTypes,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "4.3_navigation_properties",
			Version: "4.0",
			Suite:   v4_0.NavigationProperties,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "4.4_primitive_types",
			Version: "4.0",
			Suite:   v4_0.PrimitiveTypes,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "4.5_builtin_abstract_types",
			Version: "4.0",
			Suite:   v4_0.BuiltInAbstractTypes,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "4.6_annotations",
			Version: "4.0",
			Suite:   v4_0.MetadataAnnotations,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "5.1.1_primitive_data_types",
			Version: "4.0",
			Suite:   v4_0.PrimitiveDataTypes,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "5.1.1.1_numeric_edge_cases",
			Version: "4.0",
			Suite:   v4_0.NumericEdgeCases,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "5.1.1.2_byte_types",
			Version: "4.0",
			Suite:   v4_0.ByteTypes,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "5.1.1.3_int16_type",
			Version: "4.0",
			Suite:   v4_0.Int16Type,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "5.1.1.4_single_type",
			Version: "4.0",
			Suite:   v4_0.SingleType,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "5.1.1.5_numeric_boundary_tests",
			Version: "4.0",
			Suite:   v4_0.NumericBoundaryTests,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "5.1.2_nullable_properties",
			Version: "4.0",
			Suite:   v4_0.NullableProperties,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "5.1.3_collection_properties",
			Version: "4.0",
			Suite:   v4_0.CollectionProperties,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "5.1.4_temporal_data_types",
			Version: "4.0",
			Suite:   v4_0.TemporalDataTypes,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "5.1.5_guid_type",
			Version: "4.0",
			Suite:   v4_0.GuidType,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "5.1.6_binary_type",
			Version: "4.0",
			Suite:   v4_0.BinaryType,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "5.1.7_date_timeofday_types",
			Version: "4.0",
			Suite:   v4_0.DateTimeOfDayTypes,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "5.1.8_duration_type",
			Version: "4.0",
			Suite:   v4_0.DurationType,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "5.2_complex_types",
			Version: "4.0",
			Suite:   v4_0.ComplexTypes,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "5.2.1_complex_filter",
			Version: "4.0",
			Suite:   v4_0.ComplexFilter,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "5.2.2_complex_orderby",
			Version: "4.0",
			Suite:   v4_0.ComplexOrderBy,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "5.2_custom_query_options",
			Version: "4.0",
			Suite:   v4_0.CustomQueryOptions,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "5.3_enum_types",
			Version: "4.0",
			Suite:   v4_0.EnumTypes,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "5.3_enum_metadata_members",
			Version: "4.0",
			Suite:   v4_0.EnumMetadataMembers,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "5.4_type_definitions",
			Version: "4.0",
			Suite:   v4_0.TypeDefinitions,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "6.1_extensibility",
			Version: "4.0",
			Suite:   v4_0.Extensibility,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "7.1.1_unicode_strings",
			Version: "4.0",
			Suite:   v4_0.UnicodeStrings,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "8.1.1_header_content_type",
			Version: "4.0",
			Suite:   v4_0.HeaderContentType,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "8.1.2_request_headers",
			Version: "4.0",
			Suite:   v4_0.RequestHeaders,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "8.1.3_response_headers",
			Version: "4.0",
			Suite:   v4_0.ResponseHeaders,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "8.1.5_response_status_codes",
			Version: "4.0",
			Suite:   v4_0.ResponseStatusCodes,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "8.1.6_invalid_query_parameters",
			Version: "4.0",
			Suite:   v4_0.InvalidQueryParameters,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "8.1.7_method_not_allowed",
			Version: "4.0",
			Suite:   v4_0.MethodNotAllowed,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "8.2.1_cache_control_header",
			Version: "4.0",
			Suite:   v4_0.CacheControlHeader,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "8.2.2_header_if_match",
			Version: "4.0",
			Suite:   v4_0.HeaderIfMatch,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "8.2.3_header_odata_entityid",
			Version: "4.0",
			Suite:   v4_0.HeaderODataEntityId,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "8.2.4_header_content_id",
			Version: "4.0",
			Suite:   v4_0.HeaderContentId,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "8.2.5_header_location",
			Version: "4.0",
			Suite:   v4_0.HeaderLocation,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "8.2.6_header_odata_version",
			Version: "4.0",
			Suite:   v4_0.HeaderODataVersion,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "8.2.7_header_accept",
			Version: "4.0",
			Suite:   v4_0.HeaderAccept,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "8.2.8_header_prefer",
			Version: "4.0",
			Suite:   v4_0.HeaderPrefer,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "8.2.8.1_preference_allow_entityreferences",
			Version: "4.0",
			Suite:   v4_0.PreferenceAllowEntityReferences,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "8.2.8.4_preference_include_annotations",
			Version: "4.0",
			Suite:   v4_0.PreferenceIncludeAnnotations,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "8.2.9_header_maxversion",
			Version: "4.0",
			Suite:   v4_0.HeaderMaxVersion,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "8.3_error_responses",
			Version: "4.0",
			Suite:   v4_0.ErrorResponses,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "8.4_error_response_consistency",
			Version: "4.0",
			Suite:   v4_0.ErrorResponseConsistency,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "9.1_service_document",
			Version: "4.0",
			Suite:   v4_0.ServiceDocument,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "9.2_metadata_document",
			Version: "4.0",
			Suite:   v4_0.MetadataDocument,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "9.3_annotations_metadata",
			Version: "4.0",
			Suite:   v4_0.AnnotationsMetadata,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "10.1_json_format",
			Version: "4.0",
			Suite:   v4_0.JSONFormat,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "10.2_odata_annotations",
			Version: "4.0",
			Suite:   v4_0.ODataAnnotations,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.1_resource_path",
			Version: "4.0",
			Suite:   v4_0.ResourcePath,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.1_addressing_entities",
			Version: "4.0",
			Suite:   v4_0.AddressingEntities,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.2_canonical_url",
			Version: "4.0",
			Suite:   v4_0.CanonicalURL,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.3_property_access",
			Version: "4.0",
			Suite:   v4_0.PropertyAccess,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.4_collection_operations",
			Version: "4.0",
			Suite:   v4_0.CollectionOperations,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.4.1_query_search",
			Version: "4.0",
			Suite:   v4_0.QuerySearch,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.4.2_count_segment",
			Version: "4.0",
			Suite:   v4_0.CountSegment,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.5.1_query_filter",
			Version: "4.0",
			Suite:   v4_0.QueryFilter,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.5.2_query_select_orderby",
			Version: "4.0",
			Suite:   v4_0.QuerySelectOrderby,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.5.3_query_top_skip",
			Version: "4.0",
			Suite:   v4_0.QueryTopSkip,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.5.4_query_apply",
			Version: "4.0",
			Suite:   v4_0.QueryApply,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.5.4.1_advanced_apply",
			Version: "4.0",
			Suite:   v4_0.AdvancedApply,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.5.4.2_apply_transformation_catalog",
			Version: "4.0",
			Suite:   v4_0.ApplyTransformationCatalog,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.5.4.3_apply_rollup",
			Version: "4.0",
			Suite:   v4_0.QueryApplyRollup,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.5.5_query_count",
			Version: "4.0",
			Suite:   v4_0.QueryCount,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.5.6_query_expand",
			Version: "4.0",
			Suite:   v4_0.QueryExpand,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.5.7_query_skiptoken",
			Version: "4.0",
			Suite:   v4_0.QuerySkiptoken,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.5.8_parameter_aliases",
			Version: "4.0",
			Suite:   v4_0.ParameterAliases,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.5.9_nested_expand_options",
			Version: "4.0",
			Suite:   v4_0.NestedExpandOptions,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.5.9_nested_expand_advanced",
			Version: "4.0",
			Suite:   v4_0.NestedExpandAdvanced,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.5.10_query_option_combinations",
			Version: "4.0",
			Suite:   v4_0.QueryOptionCombinations,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.5.11_query_select_with_navigation_filter",
			Version: "4.0",
			Suite:   v4_0.QuerySelectWithNavigationFilter,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.5.12_pagination_edge_cases",
			Version: "4.0",
			Suite:   v4_0.PaginationEdgeCases,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.6_query_format",
			Version: "4.0",
			Suite:   v4_0.QueryFormat,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.7_metadata_levels",
			Version: "4.0",
			Suite:   v4_0.MetadataLevels,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.8_delta_links",
			Version: "4.0",
			Suite:   v4_0.DeltaLinks,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.9_lambda_operators",
			Version: "4.0",
			Suite:   v4_0.LambdaOperators,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.10_addressing_operations",
			Version: "4.0",
			Suite:   v4_0.AddressingOperations,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.11_property_value",
			Version: "4.0",
			Suite:   v4_0.PropertyValue,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.12_stream_properties",
			Version: "4.0",
			Suite:   v4_0.StreamProperties,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.13_type_casting",
			Version: "4.0",
			Suite:   v4_0.TypeCasting,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.14_url_encoding",
			Version: "4.0",
			Suite:   v4_0.URLEncoding,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.15_entity_references",
			Version: "4.0",
			Suite:   v4_0.EntityReferences,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.16_singleton_operations",
			Version: "4.0",
			Suite:   v4_0.SingletonOperations,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.17_case_sensitivity",
			Version: "4.0",
			Suite:   v4_0.CaseSensitivity,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.3.1_filter_string_functions",
			Version: "4.0",
			Suite:   v4_0.FilterStringFunctions,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.3.2_filter_date_functions",
			Version: "4.0",
			Suite:   v4_0.FilterDateFunctions,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.3.3_filter_arithmetic_functions",
			Version: "4.0",
			Suite:   v4_0.FilterArithmeticFunctions,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.3.4_filter_type_functions",
			Version: "4.0",
			Suite:   v4_0.FilterTypeFunctions,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.3.5_filter_logical_operators",
			Version: "4.0",
			Suite:   v4_0.FilterLogicalOperators,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.3.6_filter_comparison_operators",
			Version: "4.0",
			Suite:   v4_0.FilterComparisonOperators,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.3.7_filter_geo_functions",
			Version: "4.0",
			Suite:   v4_0.FilterGeoFunctions,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.3.8_filter_expanded_properties",
			Version: "4.0",
			Suite:   v4_0.FilterExpandedProperties,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.3.9_string_function_edge_cases",
			Version: "4.0",
			Suite:   v4_0.StringFunctionEdgeCases,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.3.10_filter_single_entity_navigation",
			Version: "4.0",
			Suite:   v4_0.FilterOnSingleEntityNavigationProperties,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.3.11_orderby_navigation_property",
			Version: "4.0",
			Suite:   v4_0.OrderByNavigationProperty,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.4.1_requesting_entities",
			Version: "4.0",
			Suite:   v4_0.RequestingEntities,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.4.2_create_entity",
			Version: "4.0",
			Suite:   v4_0.CreateEntity,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.4.2.1_odata_bind",
			Version: "4.0",
			Suite:   v4_0.ODataBind,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.4.3_update_entity",
			Version: "4.0",
			Suite:   v4_0.UpdateEntity,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.4.4_delete_entity",
			Version: "4.0",
			Suite:   v4_0.DeleteEntity,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.4.5_upsert",
			Version: "4.0",
			Suite:   v4_0.Upsert,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.4.6_relationships",
			Version: "4.0",
			Suite:   v4_0.Relationships,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.4.6.1_navigation_property_operations",
			Version: "4.0",
			Suite:   v4_0.NavigationPropertyOperations,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.4.7_deep_insert",
			Version: "4.0",
			Suite:   v4_0.DeepInsert,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.4.8_modify_relationships",
			Version: "4.0",
			Suite:   v4_0.ModifyRelationships,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.4.9_batch_requests",
			Version: "4.0",
			Suite:   v4_0.BatchRequests,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.4.9.1_batch_error_handling",
			Version: "4.0",
			Suite:   v4_0.BatchErrorHandling,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.4.9.3_batch_content_id_referencing",
			Version: "4.0",
			Suite:   v4_0.BatchContentIDReferencing,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.4.10_asynchronous_requests",
			Version: "4.0",
			Suite:   v4_0.AsynchronousRequests,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.4.11_head_requests",
			Version: "4.0",
			Suite:   v4_0.HEADRequests,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.4.12_returning_results",
			Version: "4.0",
			Suite:   v4_0.ReturningResults,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.4.13_action_function_parameters",
			Version: "4.0",
			Suite:   v4_0.ActionFunctionParameters,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.4.14_null_value_handling",
			Version: "4.0",
			Suite:   v4_0.NullValueHandling,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.4.15_data_validation",
			Version: "4.0",
			Suite:   v4_0.DataValidation,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.5.1_conditional_requests",
			Version: "4.0",
			Suite:   v4_0.ConditionalRequests,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.6_annotations",
			Version: "4.0",
			Suite:   v4_0.InstanceAnnotations,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "12.1_operations",
			Version: "4.0",
			Suite:   v4_0.Operations,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "13.1_asynchronous_processing",
			Version: "4.0",
			Suite:   v4_0.AsynchronousProcessing,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "14.1_vocabulary_annotations",
			Version: "4.0",
			Suite:   v4_0.VocabularyAnnotations,
		})
	}

	// Register vocabulary tests (separate from protocol versions)
	if *version == "all" || *version == "vocabularies" || *version == "vocab" {
		// Core vocabulary tests
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "vocab_core_computed",
			Version: "vocabularies",
			Suite:   core.ComputedAnnotation,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "vocab_core_immutable",
			Version: "vocabularies",
			Suite:   core.ImmutableAnnotation,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "vocab_core_description",
			Version: "vocabularies",
			Suite:   core.DescriptionAnnotation,
		})

		// Capabilities vocabulary tests
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "vocab_capabilities_insert",
			Version: "vocabularies",
			Suite:   capabilities.InsertRestrictions,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "vocab_capabilities_update",
			Version: "vocabularies",
			Suite:   capabilities.UpdateRestrictions,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "vocab_capabilities_delete",
			Version: "vocabularies",
			Suite:   capabilities.DeleteRestrictions,
		})
	}

	// Register v4.01 tests
	if *version == "all" || *version == "4.01" {
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "8.2.6_header_odata_version",
			Version: "4.01",
			Suite:   v4_01.HeaderODataVersion,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "8.2.9_header_maxversion",
			Version: "4.01",
			Suite:   v4_01.HeaderMaxVersion,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.1_key_as_segments",
			Version: "4.01",
			Suite:   v4_01.KeyAsSegments,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.5.9_nested_expand_options",
			Version: "4.01",
			Suite:   v4_01.NestedExpandOptions,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "8.2.8.6_preference_omit_values",
			Version: "4.01",
			Suite:   v4_01.PreferenceOmitValues,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "8.3.1_header_async_result",
			Version: "4.01",
			Suite:   v4_01.HeaderAsyncResult,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.5.1_filter_in_operator",
			Version: "4.01",
			Suite:   v4_01.InOperator,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.5.1.1_filter_divby_operator",
			Version: "4.01",
			Suite:   v4_01.FilterDivByOperator,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.5.8_query_compute",
			Version: "4.01",
			Suite:   v4_01.QueryCompute,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.5.11_orderby_computed_properties",
			Version: "4.01",
			Suite:   v4_01.OrderByComputedProperties,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.5.13_query_index",
			Version: "4.01",
			Suite:   v4_01.QueryIndex,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.12_query_schemaversion",
			Version: "4.01",
			Suite:   v4_01.QuerySchemaVersion,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.17_case_insensitive_system_query_options",
			Version: "4.01",
			Suite:   v4_01.CaseInsensitiveSystemQueryOptions,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "12.2_function_action_overloading",
			Version: "4.01",
			Suite:   v4_01.FunctionActionOverloading,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "19_json_batch",
			Version: "4.01",
			Suite:   v4_01.JSONBatch,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.2.5.14_wildcard_select_expand",
			Version: "4.01",
			Suite:   v4_01.WildcardSelectExpand,
		})
		testSuites = append(testSuites, TestSuiteInfo{
			Name:    "11.5.3.3_filter_matches_pattern",
			Version: "4.01",
			Suite:   v4_01.MatchesPatternFilter,
		})
	}

	if len(testSuites) == 0 {
		fmt.Println("No test suites found for version:", *version)
		os.Exit(1)
	}

	// capabilityRequirements maps a suite name to the capabilities it depends on.
	// If the service declares any of them unsupported (via Capabilities.V1 annotations in
	// $metadata), the suite is skipped rather than run-and-failed.
	// Entity-set requirements use "Products" — the primary fixture for all protocol suites.
	capabilityRequirements := map[string][]framework.RequiredCapability{
		// --- filter ---
		"11.2.5.1_query_filter":                         {framework.Require(framework.CapFilter, "Products")},
		"5.2.1_complex_filter":                          {framework.Require(framework.CapFilter, "Products")},
		"11.3.1_filter_string_functions":                {framework.Require(framework.CapFilter, "Products")},
		"11.3.2_filter_date_functions":                  {framework.Require(framework.CapFilter, "Products")},
		"11.3.3_filter_arithmetic_functions":            {framework.Require(framework.CapFilter, "Products")},
		"11.3.4_filter_type_functions":                  {framework.Require(framework.CapFilter, "Products")},
		"11.3.5_filter_logical_operators":               {framework.Require(framework.CapFilter, "Products")},
		"11.3.6_filter_comparison_operators":            {framework.Require(framework.CapFilter, "Products")},
		"11.3.7_filter_geo_functions":                   {framework.Require(framework.CapFilter, "Products")},
		"11.3.8_filter_expanded_properties":             {framework.Require(framework.CapFilter, "Products"), framework.Require(framework.CapExpand, "Products")},
		"11.3.9_string_function_edge_cases":             {framework.Require(framework.CapFilter, "Products")},
		"11.3.10_filter_single_entity_navigation":       {framework.Require(framework.CapFilter, "Products")},
		"11.2.9_lambda_operators":                       {framework.Require(framework.CapFilter, "Products")},
		"11.2.5.11_query_select_with_navigation_filter": {framework.Require(framework.CapFilter, "Products")},
		// v4.01 filter
		"11.2.5.1_filter_in_operator":  {framework.Require(framework.CapFilter, "Products")},
		"11.5.1.1_filter_divby_operator": {framework.Require(framework.CapFilter, "Products")},
		"11.5.3.3_filter_matches_pattern": {framework.Require(framework.CapFilter, "Products")},
		// --- sort ---
		"11.2.5.2_query_select_orderby":         {framework.Require(framework.CapSort, "Products")},
		"5.2.2_complex_orderby":                  {framework.Require(framework.CapSort, "Products")},
		"11.3.11_orderby_navigation_property":    {framework.Require(framework.CapSort, "Products")},
		"11.2.5.11_orderby_computed_properties":  {framework.Require(framework.CapSort, "Products")},
		// --- expand ---
		"11.2.5.6_query_expand":          {framework.Require(framework.CapExpand, "Products")},
		"11.2.5.9_nested_expand_options": {framework.Require(framework.CapExpand, "Products")},
		"11.2.5.9_nested_expand_advanced": {framework.Require(framework.CapExpand, "Products")},
		// --- count ---
		"11.2.5.5_query_count":   {framework.Require(framework.CapCount, "Products")},
		"11.2.4.2_count_segment": {framework.Require(framework.CapCount, "Products")},
		// --- search ---
		"11.2.4.1_query_search": {framework.Require(framework.CapSearch, "Products")},
		// --- top / skip ---
		"11.2.5.3_query_top_skip":        {framework.Require(framework.CapTop, "Products"), framework.Require(framework.CapSkip, "Products")},
		"11.2.5.7_query_skiptoken":        {framework.Require(framework.CapSkip, "Products")},
		"11.2.5.12_pagination_edge_cases": {framework.Require(framework.CapTop, "Products"), framework.Require(framework.CapSkip, "Products")},
		// --- insert ---
		"11.4.2_create_entity":                    {framework.Require(framework.CapInsert, "Products")},
		"11.4.2.1_odata_bind":                     {framework.Require(framework.CapInsert, "Products")},
		"11.4.7_deep_insert":                      {framework.Require(framework.CapInsert, "Products")},
		"11.4.6.1_navigation_property_operations": {framework.Require(framework.CapInsert, "Products")},
		// --- update ---
		"11.4.3_update_entity":     {framework.Require(framework.CapUpdate, "Products")},
		"11.4.8_modify_relationships": {framework.Require(framework.CapUpdate, "Products")},
		// --- upsert (insert + update) ---
		"11.4.5_upsert": {framework.Require(framework.CapInsert, "Products"), framework.Require(framework.CapUpdate, "Products")},
		// --- delete ---
		"11.4.4_delete_entity": {framework.Require(framework.CapDelete, "Products")},
		// --- batch ---
		"11.4.9_batch_requests":              {framework.Require(framework.CapBatch, "")},
		"11.4.9.1_batch_error_handling":      {framework.Require(framework.CapBatch, "")},
		"11.4.9.3_batch_content_id_referencing": {framework.Require(framework.CapBatch, "")},
		"19_json_batch":                      {framework.Require(framework.CapBatch, "")},
		// --- compute (v4.01) ---
		"11.2.5.8_query_compute": {framework.Require(framework.CapCompute, "")},
	}

	// Prepare suites (apply pattern filter) so we can compute totals for concise progress output
	type preparedSuite struct {
		info          TestSuiteInfo
		suite         *framework.TestSuite
		versionPrefix string
	}

	var suitesToRun []preparedSuite
	totalPlannedTests := 0

	for _, suiteInfo := range testSuites {
		if *pattern != "" && !strings.Contains(suiteInfo.Name, *pattern) {
			continue
		}

		suite := suiteInfo.Suite()
		suite.ServerURL = *serverURL
		suite.Debug = *debug
		suite.Verbose = *verbose
		suite.Quiet = !*verbose
		suite.Capabilities = capProfile
		suite.Strict = *strict
		if reqs, ok := capabilityRequirements[suiteInfo.Name]; ok {
			suite.RequiredCapabilities = reqs
		}

		versionPrefix := "V4"
		if suiteInfo.Version == "4.01" {
			versionPrefix = "V4.01"
		}

		totalPlannedTests += len(suite.Tests)

		suitesToRun = append(suitesToRun, preparedSuite{
			info:          suiteInfo,
			suite:         suite,
			versionPrefix: versionPrefix,
		})
	}

	if len(suitesToRun) == 0 {
		fmt.Println("No test suites matched the provided pattern.")
		os.Exit(1)
	}

	// Run tests
	fmt.Println("═════════════════════════════════════════════════════════")
	fmt.Println()

	totalSuites := len(suitesToRun)
	passedSuites := 0
	totalTests := 0
	passedTests := 0
	failedTests := 0
	skippedTests := 0

	// Collect all failed tests for final summary
	type FailedTestInfo struct {
		SuiteName string
		TestName  string
		Error     string
	}
	var allFailedTests []FailedTestInfo

	if !*verbose {
		fmt.Printf("Running %d suites (%d total tests)\n", totalSuites, totalPlannedTests)
		fmt.Println()
	}

	for idx, prepared := range suitesToRun {
		suite := prepared.suite

		if *verbose {
			fmt.Printf("\033[0;34mRunning: [%s] %s\033[0m\n", prepared.versionPrefix, prepared.info.Name)
			fmt.Println("─────────────────────────────────────────────────────────")
		}

		err := suite.Run()

		totalTests += suite.Results.Total
		passedTests += suite.Results.Passed
		failedTests += suite.Results.Failed
		skippedTests += suite.Results.Skipped

		// Collect failed tests from this suite
		for _, detail := range suite.Results.Details {
			if detail.Status == framework.StatusFail {
				allFailedTests = append(allFailedTests, FailedTestInfo{
					SuiteName: prepared.info.Name,
					TestName:  detail.Name,
					Error:     detail.Error,
				})
			}
		}

		if err == nil {
			passedSuites++
		}

		if *verbose {
			fmt.Println()
		} else {
			progressLine := fmt.Sprintf(
				"Progress: suites %d/%d | tests %d/%d | passed %d | failed %d | skipped %d",
				idx+1, totalSuites, totalTests, totalPlannedTests, passedTests, failedTests, skippedTests,
			)
			fmt.Printf("\r%-80s", progressLine)
		}
	}

	if !*verbose {
		fmt.Println()
		fmt.Println()
	}

	// Print overall summary
	fmt.Println("═════════════════════════════════════════════════════════")
	fmt.Println()
	fmt.Println("╔════════════════════════════════════════════════════════╗")
	fmt.Println("║                  OVERALL SUMMARY                       ║")
	fmt.Println("╚════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Printf("Test Scripts: %d/%d passed (%.0f%%)\n", passedSuites, totalSuites,
		float64(passedSuites)/float64(totalSuites)*100)
	fmt.Println("Individual Tests:")
	fmt.Printf("  - Total: %d\n", totalTests)
	fmt.Printf("  - Passing: %d\n", passedTests)
	fmt.Printf("  - Failing: %d\n", failedTests)
	fmt.Printf("  - Skipped: %d\n", skippedTests)
	if totalTests > 0 {
		fmt.Printf("  - Pass Rate: %.0f%%\n", float64(passedTests)/float64(totalTests)*100)
	}
	fmt.Println()

	// Print list of failed tests if any
	if len(allFailedTests) > 0 {
		fmt.Println("Failed Tests:")
		for _, failed := range allFailedTests {
			fmt.Printf("  ✗ [%s] %s\n", failed.SuiteName, failed.TestName)
			if failed.Error != "" {
				fmt.Printf("    Error: %s\n", failed.Error)
			}
		}
		fmt.Println()
	}

	// Clean exit with proper status code
	var exitCode int
	if passedSuites == totalSuites {
		fmt.Println("\033[0;32m✓ ALL TESTS PASSED\033[0m")
		fmt.Println()
		exitCode = 0
	} else {
		fmt.Println("\033[0;31m✗ SOME TESTS FAILED\033[0m")
		fmt.Println()
		exitCode = 1
	}

	os.Exit(exitCode)
}

// waitForServer polls the service root until it responds with HTTP 200 or the
// timeout (in seconds) elapses. It returns true once the server is reachable.
func waitForServer(serverURL string, timeoutSeconds int) bool {
	fmt.Printf("Waiting for OData service at %s ...\n", serverURL)
	for i := 0; i < timeoutSeconds; i++ {
		if checkServerConnectivity(serverURL) {
			fmt.Println("\033[0;32m✓ Service is reachable!\033[0m")
			fmt.Println()
			return true
		}
		time.Sleep(1 * time.Second)
	}
	return false
}

// fetchCapabilityProfile fetches $metadata from the service and parses it into a
// CapabilityProfile. Returns an error if the document cannot be retrieved or parsed.
func fetchCapabilityProfile(serverURL string) (*framework.CapabilityProfile, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(serverURL + "/$metadata")
	if err != nil {
		return nil, fmt.Errorf("GET /$metadata: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET /$metadata returned status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading $metadata body: %w", err)
	}
	return framework.ParseCapabilityProfile(body)
}

// checkServerConnectivity returns true if the service root responds with 200.
func checkServerConnectivity(serverURL string) bool {
	resp, err := framework.NewTestSuite("", "", "").Client.Get(serverURL + "/")
	if err != nil {
		return false
	}
	//nolint:errcheck
	defer func() { _ = resp.Body.Close() }()
	return resp.StatusCode == 200
}
