package v4_0

import (
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/nlstn/odata-compliance-suite/framework"
)

type csdlDocument struct {
	DataServices struct {
		Schemas []csdlSchema `xml:"Schema"`
	} `xml:"DataServices"`
}

type csdlSchema struct {
	Namespace        string                `xml:"Namespace,attr"`
	EntityTypes      []csdlEntityType      `xml:"EntityType"`
	EntityContainers []csdlEntityContainer `xml:"EntityContainer"`
	Functions        []csdlOperation       `xml:"Function"`
	Actions          []csdlOperation       `xml:"Action"`
	Annotations      []csdlAnnotations     `xml:"Annotations"`
}

type csdlEntityContainer struct {
	EntitySets []csdlEntitySet `xml:"EntitySet"`
}

type csdlEntitySet struct {
	Name       string `xml:"Name,attr"`
	EntityType string `xml:"EntityType,attr"`
}

type csdlEntityType struct {
	Name                  string                  `xml:"Name,attr"`
	BaseType              string                  `xml:"BaseType,attr"`
	Properties            []csdlProperty          `xml:"Property"`
	NavigationProperties  []csdlNavigationProperty `xml:"NavigationProperty"`
	HasStream             string                  `xml:"HasStream,attr"`
}

type csdlProperty struct {
	Name string `xml:"Name,attr"`
	Type string `xml:"Type,attr"`
}

type csdlNavigationProperty struct {
	Name string `xml:"Name,attr"`
	Type string `xml:"Type,attr"`
}

type csdlAnnotations struct {
	Target      string              `xml:"Target,attr"`
	Annotations []csdlAnnotation    `xml:"Annotation"`
}

type csdlAnnotation struct {
	Term       string           `xml:"Term,attr"`
	Collection csdlCollection  `xml:"Collection"`
}

type csdlCollection struct {
	PropertyPaths []string `xml:"PropertyPath"`
}

type csdlOperation struct {
	Name    string `xml:"Name,attr"`
	IsBound bool   `xml:"IsBound,attr"`
}

type csdlOperationsSchema struct {
	Functions []csdlOperation `xml:"Function"`
	Actions   []csdlOperation `xml:"Action"`
}

type csdlOperationDocument struct {
	DataServices struct {
		Schemas []csdlOperationsSchema `xml:"Schema"`
	} `xml:"DataServices"`
}

func getMetadataDocument(ctx *framework.TestContext) (*csdlDocument, error) {
	resp, err := ctx.GET("/$metadata")
	if err != nil {
		return nil, err
	}
	if err := ctx.AssertStatusCode(resp, 200); err != nil {
		return nil, err
	}

	var doc csdlDocument
	if err := xml.Unmarshal(resp.Body, &doc); err != nil {
		return nil, fmt.Errorf("parse metadata: %w", err)
	}
	return &doc, nil
}

func entitySetDeclared(ctx *framework.TestContext, entitySet string) (bool, error) {
	doc, err := getMetadataDocument(ctx)
	if err != nil {
		return false, err
	}
	for _, schema := range doc.DataServices.Schemas {
		for _, container := range schema.EntityContainers {
			for _, set := range container.EntitySets {
				if set.Name == entitySet {
					return true, nil
				}
			}
		}
	}
	return false, nil
}

func entityTypeDeclared(ctx *framework.TestContext, entityTypeName string) (bool, error) {
	doc, err := getMetadataDocument(ctx)
	if err != nil {
		return false, err
	}
	for _, schema := range doc.DataServices.Schemas {
		for _, entityType := range schema.EntityTypes {
			if entityType.Name == entityTypeName || schema.Namespace+"."+entityType.Name == entityTypeName {
				return true, nil
			}
		}
	}
	return false, nil
}

func entityTypeForEntitySet(ctx *framework.TestContext, entitySetName string) (string, error) {
	doc, err := getMetadataDocument(ctx)
	if err != nil {
		return "", err
	}
	for _, schema := range doc.DataServices.Schemas {
		for _, container := range schema.EntityContainers {
			for _, set := range container.EntitySets {
				if set.Name == entitySetName {
					return set.EntityType, nil
				}
			}
		}
	}
	return "", nil
}

func entityTypeHasProperty(ctx *framework.TestContext, entityTypeName, propertyName string) (bool, error) {
	doc, err := getMetadataDocument(ctx)
	if err != nil {
		return false, err
	}
	entityType, _, ok := findCsdlEntityType(doc, entityTypeName)
	if !ok {
		return false, nil
	}
	for _, property := range entityType.Properties {
		if property.Name == propertyName {
			return true, nil
		}
	}
	return false, nil
}

func entityTypeHasNavigationProperty(ctx *framework.TestContext, entityTypeName, navPropertyName string) (bool, error) {
	doc, err := getMetadataDocument(ctx)
	if err != nil {
		return false, err
	}
	entityType, _, ok := findCsdlEntityType(doc, entityTypeName)
	if !ok {
		return false, nil
	}
	for _, nav := range entityType.NavigationProperties {
		if nav.Name == navPropertyName {
			return true, nil
		}
	}
	return false, nil
}

func metadataHasNavigationPath(ctx *framework.TestContext, entityTypeName string, path []string) (bool, error) {
	doc, err := getMetadataDocument(ctx)
	if err != nil {
		return false, err
	}
	currentTypeName := entityTypeName
	for _, segment := range path {
		entityType, _, ok := findCsdlEntityType(doc, currentTypeName)
		if !ok {
			return false, nil
		}
		found := false
		for _, nav := range entityType.NavigationProperties {
			if nav.Name == segment {
				found = true
				currentTypeName = nav.Type
				if strings.HasPrefix(currentTypeName, "Collection(") && strings.HasSuffix(currentTypeName, ")") {
					currentTypeName = strings.TrimSuffix(strings.TrimPrefix(currentTypeName, "Collection("), ")")
				}
				break
			}
		}
		if !found {
			return false, nil
		}
	}
	return true, nil
}

func entityTypeHasStreamProperty(ctx *framework.TestContext, entityTypeName, propertyName string) (bool, error) {
	doc, err := getMetadataDocument(ctx)
	if err != nil {
		return false, err
	}
	entityType, _, ok := findCsdlEntityType(doc, entityTypeName)
	if !ok {
		return false, nil
	}
	if strings.EqualFold(entityType.HasStream, "true") {
		return true, nil
	}
	for _, property := range entityType.Properties {
		if property.Name == propertyName && strings.EqualFold(property.Type, "Edm.Stream") {
			return true, nil
		}
	}
	return false, nil
}

func findCsdlEntityType(doc *csdlDocument, entityTypeName string) (csdlEntityType, string, bool) {
	for _, schema := range doc.DataServices.Schemas {
		for _, entityType := range schema.EntityTypes {
			if entityType.Name == entityTypeName || schema.Namespace+"."+entityType.Name == entityTypeName {
				return entityType, schema.Namespace, true
			}
		}
	}
	return csdlEntityType{}, "", false
}

func derivedTypeDeclared(ctx *framework.TestContext, baseTypeName, derivedTypeName string) (bool, error) {
	doc, err := getMetadataDocument(ctx)
	if err != nil {
		return false, err
	}
	for _, schema := range doc.DataServices.Schemas {
		for _, entityType := range schema.EntityTypes {
			if entityType.Name != derivedTypeName && schema.Namespace+"."+entityType.Name != derivedTypeName {
				continue
			}
			if entityType.BaseType == baseTypeName || entityType.BaseType == schema.Namespace+"."+baseTypeName {
				return true, nil
			}
		}
	}
	return false, nil
}

func namespaceForEntityType(ctx *framework.TestContext, entityTypeName string) (string, error) {
	doc, err := getMetadataDocument(ctx)
	if err != nil {
		return "", err
	}
	_, namespace, ok := findCsdlEntityType(doc, entityTypeName)
	if !ok {
		return "", nil
	}
	return namespace, nil
}

func operationDeclaredInMetadata(ctx *framework.TestContext, name, kind string, bound bool) (bool, error) {
	doc, err := getMetadataDocument(ctx)
	if err != nil {
		return false, err
	}
	for _, schema := range doc.DataServices.Schemas {
		for _, op := range schema.Functions {
			if strings.EqualFold(kind, "function") && op.Name == name && op.IsBound == bound {
				return true, nil
			}
		}
		for _, op := range schema.Actions {
			if strings.EqualFold(kind, "action") && op.Name == name && op.IsBound == bound {
				return true, nil
			}
		}
	}
	return false, nil
}
