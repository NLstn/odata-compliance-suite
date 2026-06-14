package framework

import "fmt"

// Capability identifies an OData feature that can be declared unsupported via the
// Org.OData.Capabilities.V1 vocabulary.
type Capability string

const (
	CapFilter  Capability = "filter"
	CapSort    Capability = "sort"
	CapExpand  Capability = "expand"
	CapCount   Capability = "count"
	CapSearch  Capability = "search"
	CapInsert  Capability = "insert"
	CapUpdate  Capability = "update"
	CapDelete  Capability = "delete"
	CapTop     Capability = "top"
	CapSkip    Capability = "skip"
	CapBatch   Capability = "batch"
	CapCompute Capability = "compute"
)

// RequiredCapability declares that a test suite depends on a specific OData capability.
// EntitySet names the entity set for entity-set-scoped checks; leave empty for service-level.
type RequiredCapability struct {
	Cap       Capability
	EntitySet string
}

// Require is a convenience constructor for RequiredCapability.
func Require(cap Capability, entitySet string) RequiredCapability {
	return RequiredCapability{Cap: cap, EntitySet: entitySet}
}

// CapabilityProfile records which capabilities a service has declared supported or unsupported
// via Org.OData.Capabilities.V1 annotations in its $metadata document.
// Absent annotations are treated as supported (fail-open).
type CapabilityProfile struct {
	entitySetCaps map[string]map[Capability]bool
	serviceCaps   map[Capability]bool
}

// NewCapabilityProfile returns an empty profile (all capabilities presumed supported).
func NewCapabilityProfile() *CapabilityProfile {
	return &CapabilityProfile{
		entitySetCaps: make(map[string]map[Capability]bool),
		serviceCaps:   make(map[Capability]bool),
	}
}

// SetEntitySetCap records a capability declaration for a specific entity set.
func (p *CapabilityProfile) SetEntitySetCap(entitySet string, cap Capability, supported bool) {
	if p.entitySetCaps[entitySet] == nil {
		p.entitySetCaps[entitySet] = make(map[Capability]bool)
	}
	p.entitySetCaps[entitySet][cap] = supported
}

// SetServiceCap records a service-level capability declaration.
func (p *CapabilityProfile) SetServiceCap(cap Capability, supported bool) {
	p.serviceCaps[cap] = supported
}

// Supports returns whether the capability in req is declared supported.
// For entity-set-scoped requirements it checks the entity set first, then falls back to
// the service level. Absent annotations are treated as supported.
func (p *CapabilityProfile) Supports(req RequiredCapability) bool {
	if req.EntitySet != "" {
		if caps, ok := p.entitySetCaps[req.EntitySet]; ok {
			if v, ok := caps[req.Cap]; ok {
				return v
			}
		}
		// Fall back to a service-level declaration for this capability.
	}
	if v, ok := p.serviceCaps[req.Cap]; ok {
		return v
	}
	return true
}

// SkipReason returns a human-readable explanation of why a test suite is being skipped.
func (p *CapabilityProfile) SkipReason(req RequiredCapability) string {
	if req.EntitySet == "" {
		return fmt.Sprintf("service declared Org.OData.Capabilities.V1 %s=false", req.Cap)
	}
	return fmt.Sprintf("entity set %q declared Org.OData.Capabilities.V1 %s=false", req.EntitySet, req.Cap)
}
