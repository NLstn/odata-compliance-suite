package framework

import (
	"encoding/xml"
	"fmt"
	"strings"
)

// ParseCapabilityProfile parses the raw bytes of a $metadata XML document and returns
// a CapabilityProfile derived from Org.OData.Capabilities.V1 annotations.
// Absent annotations are treated as supported (fail-open); callers should do the same
// on error.
func ParseCapabilityProfile(metadataXML []byte) (*CapabilityProfile, error) {
	var doc capMetaDoc
	if err := xml.Unmarshal(metadataXML, &doc); err != nil {
		return nil, fmt.Errorf("failed to parse metadata XML: %w", err)
	}

	profile := NewCapabilityProfile()

	for _, s := range doc.DataServices.Schemas {
		containerName := s.EntityContainer.Name
		if containerName == "" {
			continue
		}
		containerTarget := s.Namespace + "." + containerName
		entitySetPrefix := containerTarget + "/"

		for _, block := range s.AnnotationBlocks {
			target := block.Target
			switch {
			case target == containerTarget:
				for _, ann := range block.Annotations {
					applyServiceCapability(profile, ann)
				}
			case strings.HasPrefix(target, entitySetPrefix):
				setName := strings.TrimPrefix(target, entitySetPrefix)
				for _, ann := range block.Annotations {
					applyEntitySetCapability(profile, setName, ann)
				}
			}
		}
	}

	return profile, nil
}

// XML struct definitions used only for capability parsing.

type capMetaDoc struct {
	DataServices capDataServices `xml:"DataServices"`
}

type capDataServices struct {
	Schemas []capSchema `xml:"Schema"`
}

type capSchema struct {
	Namespace        string           `xml:"Namespace,attr"`
	EntityContainer  capContainer     `xml:"EntityContainer"`
	AnnotationBlocks []capAnnotations `xml:"Annotations"`
}

type capContainer struct {
	Name string `xml:"Name,attr"`
}

type capAnnotations struct {
	Target      string          `xml:"Target,attr"`
	Annotations []capAnnotation `xml:"Annotation"`
}

type capAnnotation struct {
	Term   string     `xml:"Term,attr"`
	Bool   string     `xml:"Bool,attr"` // for scalar boolean terms such as TopSupported
	Record *capRecord `xml:"Record"`
}

type capRecord struct {
	PropertyValues []capPropertyValue `xml:"PropertyValue"`
}

type capPropertyValue struct {
	Property string `xml:"Property,attr"`
	Bool     string `xml:"Bool,attr"`
}

// termEntitySetCap maps a Capabilities V1 restriction term to the Capability constant
// and the record-property name that carries the boolean (e.g., "Filterable").
var termEntitySetCap = map[string]struct {
	cap      Capability
	property string
}{
	"Org.OData.Capabilities.V1.FilterRestrictions": {CapFilter, "Filterable"},
	"Org.OData.Capabilities.V1.SortRestrictions":   {CapSort, "Sortable"},
	"Org.OData.Capabilities.V1.ExpandRestrictions": {CapExpand, "Expandable"},
	"Org.OData.Capabilities.V1.CountRestrictions":  {CapCount, "Countable"},
	"Org.OData.Capabilities.V1.SearchRestrictions": {CapSearch, "Searchable"},
	"Org.OData.Capabilities.V1.InsertRestrictions": {CapInsert, "Insertable"},
	"Org.OData.Capabilities.V1.UpdateRestrictions": {CapUpdate, "Updatable"},
	"Org.OData.Capabilities.V1.DeleteRestrictions": {CapDelete, "Deletable"},
}

// termServiceCap maps scalar Capabilities V1 terms to service-level capabilities.
var termServiceCap = map[string]Capability{
	"Org.OData.Capabilities.V1.TopSupported":   CapTop,
	"Org.OData.Capabilities.V1.SkipSupported":  CapSkip,
	"Org.OData.Capabilities.V1.BatchSupported": CapBatch,
}

func applyServiceCapability(profile *CapabilityProfile, ann capAnnotation) {
	cap, ok := termServiceCap[ann.Term]
	if !ok {
		return
	}
	switch ann.Bool {
	case "false":
		profile.SetServiceCap(cap, false)
	case "true":
		profile.SetServiceCap(cap, true)
	}
}

func applyEntitySetCapability(profile *CapabilityProfile, setName string, ann capAnnotation) {
	entry, ok := termEntitySetCap[ann.Term]
	if !ok || ann.Record == nil {
		return
	}
	for _, pv := range ann.Record.PropertyValues {
		if pv.Property == entry.property {
			switch pv.Bool {
			case "false":
				profile.SetEntitySetCap(setName, entry.cap, false)
			case "true":
				profile.SetEntitySetCap(setName, entry.cap, true)
			}
		}
	}
}
