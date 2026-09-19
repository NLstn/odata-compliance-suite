package v4_0

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// MetadataReferenceIntegrity adds model-level checks around the CSDL metadata
// document. The existing metadata suite validates that the XML is present and
// well formed; these tests verify that the references inside the model resolve
// consistently and that declared members are structurally unambiguous.
func MetadataReferenceIntegrity() *framework.TestSuite {
	suite := framework.NewTestSuite(
		"9.2.1 Metadata Reference Integrity",
		"Validates that CSDL entity-set, singleton, property, navigation, enum, and container references are internally consistent.",
		"https://docs.oasis-open.org/odata/odata-csdl-xml/v4.0/odata-csdl-xml-v4.0.html#sec_CSDLElements",
	)

	suite.AddTest(
		"test_entity_set_entity_types_resolve",
		"Every EntitySet EntityType reference resolves to an EntityType declared in the metadata document",
		func(ctx *framework.TestContext) error {
			metadata, _, err := loadMetadataDocument(ctx)
			if err != nil {
				return err
			}
			types, aliases := metadataTypeIndex(metadata)
			for _, schema := range metadata.DataServices.Schemas {
				if schema.EntityContainer == nil {
					continue
				}
				for _, set := range schema.EntityContainer.EntitySets {
					if strings.TrimSpace(set.Name) == "" {
						return framework.NewError("EntitySet is missing its Name attribute")
					}
					if strings.TrimSpace(set.EntityType) == "" {
						return fmt.Errorf("EntitySet %q is missing its EntityType attribute", set.Name)
					}
					if !resolveMetadataType(set.EntityType, types, aliases) {
						return fmt.Errorf("EntitySet %q references undeclared EntityType %q", set.Name, set.EntityType)
					}
				}
			}
			return nil
		},
	)

	suite.AddTest(
		"test_singleton_types_resolve",
		"Every Singleton Type reference resolves to a declared structured type",
		func(ctx *framework.TestContext) error {
			metadata, _, err := loadMetadataDocument(ctx)
			if err != nil {
				return err
			}
			types, aliases := metadataTypeIndex(metadata)
			for _, schema := range metadata.DataServices.Schemas {
				if schema.EntityContainer == nil {
					continue
				}
				for _, singleton := range schema.EntityContainer.Singletons {
					if strings.TrimSpace(singleton.Name) == "" {
						return framework.NewError("Singleton is missing its Name attribute")
					}
					if strings.TrimSpace(singleton.Type) == "" {
						return fmt.Errorf("Singleton %q is missing its Type attribute", singleton.Name)
					}
					if !resolveMetadataType(singleton.Type, types, aliases) {
						return fmt.Errorf("Singleton %q references undeclared type %q", singleton.Name, singleton.Type)
					}
				}
			}
			return nil
		},
	)

	suite.AddTest(
		"test_container_resource_names_are_unique",
		"EntitySet and Singleton names are unique within each EntityContainer",
		func(ctx *framework.TestContext) error {
			metadata, _, err := loadMetadataDocument(ctx)
			if err != nil {
				return err
			}
			for _, schema := range metadata.DataServices.Schemas {
				if schema.EntityContainer == nil {
					continue
				}
				seen := map[string]string{}
				for _, set := range schema.EntityContainer.EntitySets {
					name := strings.TrimSpace(set.Name)
					if name == "" {
						return framework.NewError("EntitySet has an empty Name attribute")
					}
					if previous, ok := seen[name]; ok {
						return fmt.Errorf("EntityContainer %q declares resource name %q twice (%s and EntitySet)", schema.EntityContainer.Name, name, previous)
					}
					seen[name] = "EntitySet"
				}
				for _, singleton := range schema.EntityContainer.Singletons {
					name := strings.TrimSpace(singleton.Name)
					if name == "" {
						return framework.NewError("Singleton has an empty Name attribute")
					}
					if previous, ok := seen[name]; ok {
						return fmt.Errorf("EntityContainer %q declares resource name %q twice (%s and Singleton)", schema.EntityContainer.Name, name, previous)
					}
					seen[name] = "Singleton"
				}
			}
			return nil
		},
	)

	suite.AddTest(
		"test_structural_properties_are_unambiguous",
		"EntityType structural property names are non-empty, unique, and use valid Nullable values",
		func(ctx *framework.TestContext) error {
			metadata, _, err := loadMetadataDocument(ctx)
			if err != nil {
				return err
			}
			for _, schema := range metadata.DataServices.Schemas {
				for _, entityType := range schema.EntityTypes {
					seen := map[string]bool{}
					for _, property := range entityType.Properties {
						name := strings.TrimSpace(property.Name)
						if name == "" {
							return fmt.Errorf("EntityType %q contains a Property without a Name", entityType.Name)
						}
						if seen[name] {
							return fmt.Errorf("EntityType %q declares structural Property %q more than once", entityType.Name, name)
						}
						seen[name] = true
						if property.Nullable != "" && property.Nullable != "true" && property.Nullable != "false" && property.Nullable != "1" && property.Nullable != "0" {
							return fmt.Errorf("EntityType %q Property %q has invalid Nullable value %q", entityType.Name, name, property.Nullable)
						}
					}
				}
			}
			return nil
		},
	)

	suite.AddTest(
		"test_navigation_properties_are_unambiguous",
		"EntityType navigation property names are non-empty and do not collide with structural properties",
		func(ctx *framework.TestContext) error {
			metadata, _, err := loadMetadataDocument(ctx)
			if err != nil {
				return err
			}
			for _, schema := range metadata.DataServices.Schemas {
				for _, entityType := range schema.EntityTypes {
					structural := map[string]bool{}
					for _, property := range entityType.Properties {
						structural[property.Name] = true
					}
					seen := map[string]bool{}
					for _, navigation := range entityType.NavigationProperties {
						name := strings.TrimSpace(navigation.Name)
						if name == "" {
							return fmt.Errorf("EntityType %q contains a NavigationProperty without a Name", entityType.Name)
						}
						if structural[name] {
							return fmt.Errorf("EntityType %q uses %q for both Property and NavigationProperty", entityType.Name, name)
						}
						if seen[name] {
							return fmt.Errorf("EntityType %q declares NavigationProperty %q more than once", entityType.Name, name)
						}
						seen[name] = true
					}
				}
			}
			return nil
		},
	)

	suite.AddTest(
		"test_enum_members_are_unique",
		"Each EnumType declares non-empty, unique member names and valid numeric values when supplied",
		func(ctx *framework.TestContext) error {
			metadata, _, err := loadMetadataDocument(ctx)
			if err != nil {
				return err
			}
			for _, schema := range metadata.DataServices.Schemas {
				for _, enumType := range schema.EnumTypes {
					seen := map[string]bool{}
					for _, member := range enumType.Members {
						name := strings.TrimSpace(member.Name)
						if name == "" {
							return fmt.Errorf("EnumType %q contains a Member without a Name", enumType.Name)
						}
						if seen[name] {
							return fmt.Errorf("EnumType %q declares Member %q more than once", enumType.Name, name)
						}
						seen[name] = true
						if member.Value != "" {
							if _, err := parseEnumValue(member.Value); err != nil {
								return fmt.Errorf("EnumType %q Member %q has invalid Value %q: %w", enumType.Name, name, member.Value, err)
							}
						}
					}
				}
			}
			return nil
		},
	)

	return suite
}

type metadataNavigationProperty struct {
	Name string `xml:"Name,attr"`
}

type metadataEnumMember struct {
	Name  string `xml:"Name,attr"`
	Value string `xml:"Value,attr"`
}

type metadataEnumType struct {
	Name    string               `xml:"Name,attr"`
	Members []metadataEnumMember `xml:"Member"`
}

func metadataTypeIndex(metadata *csdlMetadataDocument) (map[string]bool, map[string]string) {
	types := map[string]bool{}
	aliases := map[string]string{}
	for _, schema := range metadata.DataServices.Schemas {
		if schema.Alias != "" {
			aliases[schema.Alias] = schema.Namespace
		}
		for _, entityType := range schema.EntityTypes {
			types[schema.Namespace+"."+entityType.Name] = true
		}
		for _, complexType := range schema.ComplexTypes {
			types[schema.Namespace+"."+complexType.Name] = true
		}
	}
	return types, aliases
}

func resolveMetadataType(typeName string, types map[string]bool, aliases map[string]string) bool {
	typeName = strings.TrimSpace(typeName)
	if strings.HasPrefix(typeName, "Collection(") && strings.HasSuffix(typeName, ")") {
		typeName = strings.TrimSuffix(strings.TrimPrefix(typeName, "Collection("), ")")
	}
	if types[typeName] {
		return true
	}
	if dot := strings.Index(typeName, "."); dot > 0 {
		if namespace, ok := aliases[typeName[:dot]]; ok {
			return types[namespace+typeName[dot:]]
		}
		return false
	}
	for qualified := range types {
		if strings.HasSuffix(qualified, "."+typeName) {
			return true
		}
	}
	return false
}

func parseEnumValue(value string) (int64, error) {
	return strconv.ParseInt(strings.TrimSpace(value), 10, 64)
}
