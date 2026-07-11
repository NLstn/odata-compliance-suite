package framework

import (
	"encoding/xml"
	"fmt"
	"strings"
)

const capabilitiesNamespace = "Org.OData.Capabilities.V1"

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
	aliases := map[string]string{}
	for _, ref := range doc.References {
		for _, include := range ref.Includes {
			if include.Alias != "" && include.Namespace != "" {
				aliases[include.Alias] = include.Namespace
			}
		}
	}
	for _, s := range doc.DataServices.Schemas {
		if s.Alias != "" && s.Namespace != "" {
			aliases[s.Alias] = s.Namespace
		}
	}

	containers := make(map[string]struct{})
	entitySetTargets := make(map[string]string)

	for _, s := range doc.DataServices.Schemas {
		containerName := s.EntityContainer.Name
		if containerName == "" {
			continue
		}
		containerTarget := s.Namespace + "." + containerName
		// Annotations may be children of the model element they annotate.
		for _, ann := range s.EntityContainer.Annotations {
			applyServiceCapability(profile, canonicalTerm(ann.Term, aliases), ann)
		}
		for _, set := range s.EntityContainer.EntitySets {
			setTarget := containerTarget + "/" + set.Name
			entitySetTargets[setTarget] = set.Name
			for _, ann := range set.Annotations {
				applyEntitySetCapability(profile, set.Name, canonicalTerm(ann.Term, aliases), ann)
			}
		}
		containers[containerTarget] = struct{}{}
	}

	// External annotation blocks need not be in the schema that declares their
	// target, so resolve them against all containers and entity sets in the model.
	for _, s := range doc.DataServices.Schemas {
		for _, block := range s.AnnotationBlocks {
			target := canonicalTarget(block.Target, aliases)
			if _, ok := containers[target]; ok {
				for _, ann := range block.Annotations {
					applyServiceCapability(profile, canonicalTerm(ann.Term, aliases), ann)
				}
				continue
			}
			if setName, ok := entitySetTargets[target]; ok {
				for _, ann := range block.Annotations {
					applyEntitySetCapability(profile, setName, canonicalTerm(ann.Term, aliases), ann)
				}
			}
		}
	}

	return profile, nil
}

// XML struct definitions used only for capability parsing.

type capMetaDoc struct {
	References   []capReference  `xml:"Reference"`
	DataServices capDataServices `xml:"DataServices"`
}

type capReference struct {
	Includes []capInclude `xml:"Include"`
}

type capInclude struct {
	Namespace string `xml:"Namespace,attr"`
	Alias     string `xml:"Alias,attr"`
}

type capDataServices struct {
	Schemas []capSchema `xml:"Schema"`
}

type capSchema struct {
	Namespace        string           `xml:"Namespace,attr"`
	Alias            string           `xml:"Alias,attr"`
	EntityContainer  capContainer     `xml:"EntityContainer"`
	AnnotationBlocks []capAnnotations `xml:"Annotations"`
}

type capContainer struct {
	Name        string          `xml:"Name,attr"`
	Annotations []capAnnotation `xml:"Annotation"`
	EntitySets  []capEntitySet  `xml:"EntitySet"`
}

type capEntitySet struct {
	Name        string          `xml:"Name,attr"`
	Annotations []capAnnotation `xml:"Annotation"`
}

type capAnnotations struct {
	Target      string          `xml:"Target,attr"`
	Annotations []capAnnotation `xml:"Annotation"`
}

type capAnnotation struct {
	Term        string     `xml:"Term,attr"`
	BoolAttr    string     `xml:"Bool,attr"`
	BoolElement string     `xml:"Bool"`
	Record      *capRecord `xml:"Record"`
}

type capRecord struct {
	PropertyValues []capPropertyValue `xml:"PropertyValue"`
}

type capPropertyValue struct {
	Property    string     `xml:"Property,attr"`
	BoolAttr    string     `xml:"Bool,attr"`
	BoolElement string     `xml:"Bool"`
	Record      *capRecord `xml:"Record"`
}

// termEntitySetCap maps a Capabilities V1 restriction term to the Capability constant
// and the record-property name that carries the boolean (e.g., "Filterable").
var termEntitySetCap = map[string]struct {
	cap      Capability
	property string
}{
	capabilitiesNamespace + ".FilterRestrictions": {CapFilter, "Filterable"},
	capabilitiesNamespace + ".SortRestrictions":   {CapSort, "Sortable"},
	capabilitiesNamespace + ".ExpandRestrictions": {CapExpand, "Expandable"},
	capabilitiesNamespace + ".CountRestrictions":  {CapCount, "Countable"},
	capabilitiesNamespace + ".SearchRestrictions": {CapSearch, "Searchable"},
	capabilitiesNamespace + ".InsertRestrictions": {CapInsert, "Insertable"},
	capabilitiesNamespace + ".UpdateRestrictions": {CapUpdate, "Updatable"},
	capabilitiesNamespace + ".DeleteRestrictions": {CapDelete, "Deletable"},
	capabilitiesNamespace + ".SelectSupport":      {CapSelect, "Supported"},
	capabilitiesNamespace + ".ReadRestrictions":   {CapRead, "Readable"},
}

// termScalarCap maps Tag/boolean Capabilities V1 terms. These terms can be
// applied at the entity-container or entity-set level.
var termScalarCap = map[string]Capability{
	capabilitiesNamespace + ".TopSupported":          CapTop,
	capabilitiesNamespace + ".SkipSupported":         CapSkip,
	capabilitiesNamespace + ".BatchSupported":        CapBatch,
	capabilitiesNamespace + ".ComputeSupported":      CapCompute,
	capabilitiesNamespace + ".KeyAsSegmentSupported": CapKeyAsSegment,
	capabilitiesNamespace + ".IndexableByKey":        CapIndexByKey,
}

func applyServiceCapability(profile *CapabilityProfile, term string, ann capAnnotation) {
	if cap, ok := termScalarCap[term]; ok {
		// These terms have Tag/boolean semantics: an annotation without an
		// explicit expression means true.
		supported, present := annotationBool(ann)
		if !present {
			supported = true
		}
		profile.SetServiceCap(cap, supported)
		return
	}

	if term == capabilitiesNamespace+".BatchSupport" && ann.Record != nil {
		if supported, ok := recordBool(ann.Record, "Supported"); ok {
			profile.SetServiceCap(CapBatch, supported)
		}
		return
	}

	if term == capabilitiesNamespace+".DefaultCapabilities" && ann.Record != nil {
		applyDefaultCapabilities(profile, ann.Record)
		return
	}

	if entry, ok := termEntitySetCap[term]; ok && ann.Record != nil {
		if supported, ok := recordBool(ann.Record, entry.property); ok {
			profile.SetServiceCap(entry.cap, supported)
		}
	}
}

func applyEntitySetCapability(profile *CapabilityProfile, setName, term string, ann capAnnotation) {
	if cap, ok := termScalarCap[term]; ok {
		supported, present := annotationBool(ann)
		if !present {
			supported = true
		}
		profile.SetEntitySetCap(setName, cap, supported)
		return
	}

	if term == capabilitiesNamespace+".BatchSupport" && ann.Record != nil {
		if supported, ok := recordBool(ann.Record, "Supported"); ok {
			profile.SetEntitySetCap(setName, CapBatch, supported)
		}
		return
	}

	entry, ok := termEntitySetCap[term]
	if !ok || ann.Record == nil {
		return
	}
	if supported, ok := recordBool(ann.Record, entry.property); ok {
		profile.SetEntitySetCap(setName, entry.cap, supported)
	}
}

func annotationBool(ann capAnnotation) (bool, bool) {
	return parseBoolExpression(ann.BoolAttr, ann.BoolElement)
}

func recordBool(record *capRecord, property string) (bool, bool) {
	for _, pv := range record.PropertyValues {
		if pv.Property == property {
			return parseBoolExpression(pv.BoolAttr, pv.BoolElement)
		}
	}
	return false, false
}

func applyDefaultCapabilities(profile *CapabilityProfile, record *capRecord) {
	for _, pv := range record.PropertyValues {
		if cap, ok := defaultScalarCap[pv.Property]; ok {
			if supported, present := parseBoolExpression(pv.BoolAttr, pv.BoolElement); present {
				profile.SetServiceCap(cap, supported)
			}
			continue
		}

		entry, ok := defaultRecordCap[pv.Property]
		if !ok || pv.Record == nil {
			continue
		}
		if supported, present := recordBool(pv.Record, entry.property); present {
			profile.SetServiceCap(entry.cap, supported)
		}
	}
}

var defaultScalarCap = map[string]Capability{
	"TopSupported":     CapTop,
	"SkipSupported":    CapSkip,
	"ComputeSupported": CapCompute,
	"IndexableByKey":   CapIndexByKey,
}

var defaultRecordCap = map[string]struct {
	cap      Capability
	property string
}{
	"FilterRestrictions": {CapFilter, "Filterable"},
	"SortRestrictions":   {CapSort, "Sortable"},
	"ExpandRestrictions": {CapExpand, "Expandable"},
	"CountRestrictions":  {CapCount, "Countable"},
	"SearchRestrictions": {CapSearch, "Searchable"},
	"InsertRestrictions": {CapInsert, "Insertable"},
	"UpdateRestrictions": {CapUpdate, "Updatable"},
	"DeleteRestrictions": {CapDelete, "Deletable"},
	"SelectSupport":      {CapSelect, "Supported"},
	"ReadRestrictions":   {CapRead, "Readable"},
}

func parseBoolExpression(attribute, element string) (bool, bool) {
	value := strings.TrimSpace(attribute)
	if value == "" {
		value = strings.TrimSpace(element)
	}
	switch value {
	case "true", "1":
		return true, true
	case "false", "0":
		return false, true
	default:
		return false, false
	}
}

func canonicalTerm(term string, aliases map[string]string) string {
	if strings.HasPrefix(term, capabilitiesNamespace+".") {
		return term
	}
	return canonicalQualifiedName(term, aliases)
}

func canonicalTarget(target string, aliases map[string]string) string {
	return canonicalQualifiedName(target, aliases)
}

func canonicalQualifiedName(name string, aliases map[string]string) string {
	for alias, namespace := range aliases {
		if strings.HasPrefix(name, alias+".") {
			return namespace + strings.TrimPrefix(name, alias)
		}
	}
	return name
}
