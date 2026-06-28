# OData Conformance Level Mapping

This document describes how test suites are mapped to OData OASIS conformance levels.

## Levels

| Level        | Description |
|--------------|-------------|
| **Minimal**  | Core requirements every OData service must satisfy: service document, metadata, basic entity read, JSON format, HTTP headers and error responses. |
| **Intermediate** | Filtering, sorting, paging, counting, and basic CRUD (create, update, delete). |
| **Advanced** | Expand, search, batch, async processing, concurrency (ETags), functions/actions, and advanced querying (apply, compute, lambda, delta). |

Conformance levels are **cumulative**: a service must meet all lower levels to claim a higher one.

## Suite → Level → Feature mapping

### Minimal

| Suite | Feature |
|-------|---------|
| `1.1_introduction` | Service Discovery |
| `2.1_conformance` | Service Discovery |
| `9.1_service_document` | Service Discovery |
| `3.1_edmx_element` | Metadata |
| `3.2_dataservices_element` | Metadata |
| `3.3_reference_element` | Metadata |
| `3.4_include_element` | Metadata |
| `3.5_includeannotations_element` | Metadata |
| `9.2_metadata_document` | Metadata |
| `9.3_annotations_metadata` | Metadata |
| `4.1_nominal_types` – `4.6_annotations` | Data Types |
| `5.1.1_primitive_data_types` – `5.4_type_definitions` | Data Types |
| `5.2_complex_types`, `5.3_enum_types`, `5.3_enum_metadata_members` | Data Types |
| `6.1_extensibility` | HTTP Protocol |
| `7.1.1_unicode_strings` | HTTP Protocol |
| `8.1.1_header_content_type` – `8.1.7_method_not_allowed` | HTTP Protocol |
| `8.2.1_cache_control_header` – `8.2.9_header_maxversion` | HTTP Protocol |
| `8.3_error_responses`, `8.4_error_response_consistency` | HTTP Protocol |
| `11.2.17_case_sensitivity`, `11.2.17_case_insensitive_system_query_options` | HTTP Protocol |
| `5.2_custom_query_options` | HTTP Protocol |
| `11.4.11_head_requests` | HTTP Protocol |
| `10.1_json_format`, `10.2_odata_annotations` | JSON Format |
| `11.2.6_query_format` | JSON Format |
| `11.1_resource_path` | Entity Read |
| `11.2.1_addressing_entities`, `11.2.1_key_as_segments` | Entity Read |
| `11.2.2_canonical_url` | Entity Read |
| `11.2.3_property_access` | Entity Read |
| `11.2.4_collection_operations` | Entity Read |
| `11.2.11_property_value` | Entity Read |
| `11.2.12_stream_properties` | Entity Read |
| `11.2.13_type_casting` | Entity Read |
| `11.2.14_url_encoding` | Entity Read |
| `11.2.16_singleton_operations` | Entity Read |
| `11.4.1_requesting_entities` | Entity Read |
| `vocab_core_description` | Vocabularies |

### Intermediate

| Suite | Feature |
|-------|---------|
| `8.2.8.1_preference_allow_entityreferences` | HTTP Protocol |
| `8.2.8.4_preference_include_annotations` | HTTP Protocol |
| `11.2.7_metadata_levels` | JSON Format |
| `11.2.10_addressing_operations` | Entity Read |
| `11.2.15_entity_references` | Entity Read |
| `11.2.5.1_query_filter`, `5.2.1_complex_filter` | Filtering |
| `11.3.1_filter_string_functions` – `11.3.6_filter_comparison_operators` | Filtering |
| `11.3.9_string_function_edge_cases` | Filtering |
| `11.3.10_filter_single_entity_navigation` | Filtering |
| `11.2.5.1_filter_in_operator` (v4.01) | Filtering |
| `11.5.1.1_filter_divby_operator` (v4.01) | Filtering |
| `11.5.3.3_filter_matches_pattern` (v4.01) | Filtering |
| `11.2.5.2_query_select_orderby`, `5.2.2_complex_orderby` | Sorting |
| `11.2.5.3_query_top_skip` | Paging |
| `11.2.5.7_query_skiptoken` | Paging |
| `11.2.5.12_pagination_edge_cases` | Paging |
| `11.2.5.5_query_count`, `11.2.4.2_count_segment` | Counting |
| `11.4.2_create_entity`, `11.4.2.1_odata_bind` | Data Modification |
| `11.4.3_update_entity` | Data Modification |
| `11.4.4_delete_entity` | Data Modification |
| `11.4.6_relationships`, `11.4.6.1_navigation_property_operations` | Data Modification |
| `11.4.8_modify_relationships` | Data Modification |
| `11.4.12_returning_results` | Data Modification |
| `11.4.14_null_value_handling` | Data Modification |
| `11.4.15_data_validation` | Data Modification |
| `11.2.5.8_parameter_aliases` | Advanced Querying |
| `11.2.5.10_query_option_combinations` | Advanced Querying |
| `vocab_capabilities_insert`, `vocab_capabilities_update`, `vocab_capabilities_delete` | Vocabularies |

### Advanced

| Suite | Feature |
|-------|---------|
| `8.2.8.6_preference_omit_values` (v4.01) | HTTP Protocol |
| `8.3.1_header_async_result` (v4.01) | HTTP Protocol |
| `11.2.12_query_schemaversion` (v4.01) | Metadata |
| `11.3.7_filter_geo_functions` | Filtering |
| `11.3.8_filter_expanded_properties` | Filtering |
| `11.2.9_lambda_operators` | Filtering |
| `11.2.5.11_query_select_with_navigation_filter` | Filtering |
| `11.3.11_orderby_navigation_property` | Sorting |
| `11.2.5.11_orderby_computed_properties` (v4.01) | Sorting |
| `11.2.4.1_query_search` | Searching |
| `11.2.5.6_query_expand` | Expanding |
| `11.2.5.9_nested_expand_options`, `11.2.5.9_nested_expand_advanced` | Expanding |
| `11.2.5.14_wildcard_select_expand` (v4.01) | Expanding |
| `11.4.5_upsert` | Data Modification |
| `11.4.7_deep_insert` | Data Modification |
| `11.4.9_batch_requests` | Batch |
| `11.4.9.1_batch_error_handling` | Batch |
| `11.4.9.2_batch_changeset_atomicity` | Batch |
| `11.4.9.3_batch_content_id_referencing` | Batch |
| `19_json_batch` (v4.01) | Batch |
| `11.2.5.4_query_apply` – `11.2.5.4.3_apply_rollup` | Advanced Querying |
| `11.2.5.8_query_compute` (v4.01) | Advanced Querying |
| `11.2.5.13_query_index` (v4.01) | Advanced Querying |
| `11.2.8_delta_links` | Advanced Querying |
| `12.1_operations`, `12.2_function_action_overloading` (v4.01) | Operations |
| `11.4.13_action_function_parameters` | Operations |
| `11.5.1_conditional_requests` | Concurrency |
| `11.4.10_asynchronous_requests` | Async |
| `13.1_asynchronous_processing` | Async |
| `11.6_annotations` | Annotations |
| `14.1_vocabulary_annotations` | Annotations |
| `vocab_core_computed`, `vocab_core_immutable` | Vocabularies |

## Sample output

```
╔════════════════════════════════════════════════════════╗
║              CONFORMANCE LEVEL REPORT                  ║
╚════════════════════════════════════════════════════════╝

  ✓ [OData 4.0] Minimal: Met (42/42 suites)
  ✓ [OData 4.0] Intermediate: Met (21/21 suites)
  ✗ [OData 4.0] Advanced: Not Met (18/20 suites)
  → Highest level fully met: OData 4.0 Intermediate

  ✓ [OData 4.01] Minimal: Met (4/4 suites)
  ✗ [OData 4.01] Intermediate: Not Met (3/4 suites)
  → Highest level fully met: OData 4.01 Minimal
```

The per-feature matrix is printed in `--verbose` mode and shows passed/failed/skipped test counts per feature group.
