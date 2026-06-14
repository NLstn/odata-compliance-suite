package framework_test

import (
	"testing"

	"github.com/nlstn/odata-compliance-suite/framework"
)

// testMetadataXML is a minimal $metadata document that declares:
//   - TopSupported=false and BatchSupported=false at the container level
//   - FilterRestrictions.Filterable=false and InsertRestrictions.Insertable=false on Products
var testMetadataXML = []byte(`<?xml version="1.0" encoding="utf-8"?>
<edmx:Edmx Version="4.0" xmlns:edmx="http://docs.oasis-open.org/odata/ns/edmx">
  <edmx:DataServices>
    <Schema Namespace="TestService" xmlns="http://docs.oasis-open.org/odata/ns/edm">
      <EntityType Name="Product">
        <Key><PropertyRef Name="ID"/></Key>
        <Property Name="ID" Type="Edm.Guid" Nullable="false"/>
      </EntityType>
      <EntityContainer Name="Container">
        <EntitySet Name="Products" EntityType="TestService.Product"/>
      </EntityContainer>
      <Annotations Target="TestService.Container">
        <Annotation Term="Org.OData.Capabilities.V1.TopSupported" Bool="false"/>
        <Annotation Term="Org.OData.Capabilities.V1.BatchSupported" Bool="false"/>
      </Annotations>
      <Annotations Target="TestService.Container/Products">
        <Annotation Term="Org.OData.Capabilities.V1.FilterRestrictions">
          <Record>
            <PropertyValue Property="Filterable" Bool="false"/>
          </Record>
        </Annotation>
        <Annotation Term="Org.OData.Capabilities.V1.InsertRestrictions">
          <Record>
            <PropertyValue Property="Insertable" Bool="false"/>
          </Record>
        </Annotation>
      </Annotations>
    </Schema>
  </edmx:DataServices>
</edmx:Edmx>`)

func TestParseCapabilityProfile(t *testing.T) {
	profile, err := framework.ParseCapabilityProfile(testMetadataXML)
	if err != nil {
		t.Fatalf("ParseCapabilityProfile: %v", err)
	}

	cases := []struct {
		req  framework.RequiredCapability
		want bool
		desc string
	}{
		// Service-level scalar terms
		{framework.Require(framework.CapTop, ""), false, "TopSupported=false (service)"},
		{framework.Require(framework.CapBatch, ""), false, "BatchSupported=false (service)"},
		// Entity-set-level record terms
		{framework.Require(framework.CapFilter, "Products"), false, "Filterable=false on Products"},
		{framework.Require(framework.CapInsert, "Products"), false, "Insertable=false on Products"},
		// Service-level fallback for entity-set-scoped request
		{framework.Require(framework.CapTop, "Products"), false, "TopSupported=false falls back from service level"},
		// Absent → treated as supported
		{framework.Require(framework.CapSort, "Products"), true, "SortRestrictions absent → supported"},
		{framework.Require(framework.CapExpand, "Products"), true, "ExpandRestrictions absent → supported"},
		{framework.Require(framework.CapDelete, "Products"), true, "DeleteRestrictions absent → supported"},
		{framework.Require(framework.CapSkip, ""), true, "SkipSupported absent → supported"},
	}

	for _, tc := range cases {
		got := profile.Supports(tc.req)
		if got != tc.want {
			t.Errorf("[%s] Supports(%v, %q) = %v, want %v", tc.desc, tc.req.Cap, tc.req.EntitySet, got, tc.want)
		}
	}
}

func TestParseCapabilityProfileAllSupported(t *testing.T) {
	// A metadata document with no Capabilities annotations → everything is supported.
	noRestrictions := []byte(`<?xml version="1.0" encoding="utf-8"?>
<edmx:Edmx Version="4.0" xmlns:edmx="http://docs.oasis-open.org/odata/ns/edmx">
  <edmx:DataServices>
    <Schema Namespace="FullService" xmlns="http://docs.oasis-open.org/odata/ns/edm">
      <EntityContainer Name="Container"/>
    </Schema>
  </edmx:DataServices>
</edmx:Edmx>`)

	profile, err := framework.ParseCapabilityProfile(noRestrictions)
	if err != nil {
		t.Fatalf("ParseCapabilityProfile: %v", err)
	}

	caps := []framework.Capability{
		framework.CapFilter, framework.CapSort, framework.CapExpand, framework.CapCount,
		framework.CapSearch, framework.CapInsert, framework.CapUpdate, framework.CapDelete,
		framework.CapTop, framework.CapSkip, framework.CapBatch, framework.CapCompute,
	}
	for _, cap := range caps {
		if !profile.Supports(framework.Require(cap, "")) {
			t.Errorf("Supports(%v, service) = false, want true (no annotation present)", cap)
		}
		if !profile.Supports(framework.Require(cap, "Products")) {
			t.Errorf("Supports(%v, Products) = false, want true (no annotation present)", cap)
		}
	}
}

func TestSkipReason(t *testing.T) {
	p := framework.NewCapabilityProfile()

	if r := p.SkipReason(framework.Require(framework.CapFilter, "Products")); r == "" {
		t.Error("SkipReason for entity-set cap should not be empty")
	}
	if r := p.SkipReason(framework.Require(framework.CapBatch, "")); r == "" {
		t.Error("SkipReason for service-level cap should not be empty")
	}
}

func TestParseSelectSupportComputeable(t *testing.T) {
	// SelectSupport.Computeable=false on an entity set should gate CapCompute.
	xml := []byte(`<?xml version="1.0" encoding="utf-8"?>
<edmx:Edmx Version="4.0" xmlns:edmx="http://docs.oasis-open.org/odata/ns/edmx">
  <edmx:DataServices>
    <Schema Namespace="ComputeService" xmlns="http://docs.oasis-open.org/odata/ns/edm">
      <EntityContainer Name="Container"/>
      <Annotations Target="ComputeService.Container/Products">
        <Annotation Term="Org.OData.Capabilities.V1.SelectSupport">
          <Record>
            <PropertyValue Property="Computeable" Bool="false"/>
          </Record>
        </Annotation>
      </Annotations>
    </Schema>
  </edmx:DataServices>
</edmx:Edmx>`)

	profile, err := framework.ParseCapabilityProfile(xml)
	if err != nil {
		t.Fatalf("ParseCapabilityProfile: %v", err)
	}
	if profile.Supports(framework.Require(framework.CapCompute, "Products")) {
		t.Error("expected CapCompute unsupported on Products (SelectSupport.Computeable=false)")
	}
	// Other caps unaffected
	if !profile.Supports(framework.Require(framework.CapFilter, "Products")) {
		t.Error("expected CapFilter still supported")
	}
}

func TestCapabilityProfileSetAndGet(t *testing.T) {
	p := framework.NewCapabilityProfile()

	p.SetEntitySetCap("Products", framework.CapFilter, false)
	p.SetServiceCap(framework.CapBatch, false)

	if p.Supports(framework.Require(framework.CapFilter, "Products")) {
		t.Error("expected filter unsupported on Products")
	}
	if p.Supports(framework.Require(framework.CapBatch, "")) {
		t.Error("expected batch unsupported at service level")
	}
	if !p.Supports(framework.Require(framework.CapSort, "Products")) {
		t.Error("expected sort supported (not set)")
	}
}
