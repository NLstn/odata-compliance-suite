# Reference Data-Model Contract

The compliance suite is a **black-box** tester: it issues HTTP requests against a
running OData service and validates the responses against the OData v4.0 / v4.01
specifications. To do that, the tests assume the service exposes a specific,
well-known data model and seed data — the **reference model** described here.

> **v1 scope:** The reference model below is a **hard requirement**. A service
> under test must expose these exact entity sets, properties, operations, and
> seed rows, or many tests will fail for reasons unrelated to spec compliance.
> Support for configurable / dynamically-discovered models is planned — see
> [Roadmap](#roadmap).

The reference implementation lives in the `go-odata` repository under
`cmd/complianceserver`. Any service (in any language) that reproduces this
contract can be measured by the suite.

---

## Service root

- The service root (`GET /`) must return **HTTP 200** and a JSON service document
  listing the entity sets and the `Company` singleton.
- `GET /$metadata` must return the CSDL document as `application/xml`, and
  `GET /$metadata?$format=json` as `application/json`.
- The schema **namespace is discovered from `$metadata`** (the reference uses
  `ComplianceService`, but tests read it dynamically — you may use your own).
- Default port in the reference server is **9090**; point `-server` at your root.

---

## Entity sets

| Entity set            | Key(s)                          | Notes                                  |
|-----------------------|---------------------------------|----------------------------------------|
| `Products`            | `ID` (Edm.Guid)                 | Primary fixture; change-tracking enabled |
| `Categories`          | `ID` (Edm.Guid)                 | Related to Products                    |
| `ProductDescriptions` | `ProductID` + `LanguageKey`     | Composite key; multilingual            |
| `MediaItems`          | `ID` (Edm.Guid)                 | Media entity (has stream)              |
| `ReadOnlyItems`       | `ID` (Edm.Guid)                 | Carries capability restrictions        |
| `DecimalSamples`      | `ID` (Edm.Guid)                 | Isolated `Edm.Decimal` fixture         |

Plus one **singleton**: `Company`.

### `Products`

| Property          | EDM type                  | Nullable | Notes                                            |
|-------------------|---------------------------|----------|--------------------------------------------------|
| `ID`              | Edm.Guid                  | no       | Key, server-generated                            |
| `Name`            | Edm.String (max 100)      | no       | Searchable; `Core.Description` annotation        |
| `Description`     | Edm.String (max 500)      | yes      | `Core.Description` annotation                    |
| `Price`           | Edm.Double                | no       | precision 10, scale 2 (note: **Double**, not Decimal) |
| `Rating`          | Edm.Byte                  | no       |                                                  |
| `Temperature`     | Edm.SByte                 | no       |                                                  |
| `Quantity`        | Edm.Int16                 | no       |                                                  |
| `Weight`          | Edm.Single                | no       |                                                  |
| `Data`            | Edm.Binary                | yes      |                                                  |
| `ReleaseDate`     | Edm.Date                  | yes      | e.g. `2024-01-15`                                |
| `OpenTime`        | Edm.TimeOfDay             | yes      | e.g. `09:30:00`                                  |
| `ShippingTime`    | Edm.Duration              | yes      | e.g. `P1D`                                        |
| `ProcessingTime`  | Edm.Duration              | yes      | e.g. `PT45S`                                      |
| `Offset`          | Edm.Duration              | yes      |                                                  |
| `CategoryID`      | Edm.Guid                  | yes      | FK to `Categories`                               |
| `Status`          | `ProductStatus` (enum, flags) | no   | see [Enum types](#enum-types)                    |
| `Version`         | Edm.Int32                 | no       | **ETag** source; increments on update            |
| `CreatedAt`       | Edm.DateTimeOffset        | no       | `Core.Computed`                                  |
| `SerialNumber`    | Edm.String (max 50)       | yes      | `Core.Immutable`, `Core.Description`             |
| `ProductType`     | Edm.String (max 50)       | no       | Discriminator (`Product` / `SpecialProduct`)     |
| `SpecialProperty` | Edm.String (max 200)      | yes      | Present on `SpecialProduct` derived type         |
| `SpecialFeature`  | Edm.String (max 100)      | yes      | Present on `SpecialProduct` derived type         |
| `ShippingAddress` | `Address` (complex)       | yes      |                                                  |
| `Dimensions`      | `Dimensions` (complex)    | yes      |                                                  |
| `Location`        | Edm.GeographyPoint        | yes      | WKT in reference                                 |
| `Route`           | Edm.GeographyLineString   | yes      |                                                  |
| `Area`            | Edm.GeographyPolygon      | yes      |                                                  |
| `Photo`           | Edm.Stream                | —        | Stream property                                  |

Navigation properties:
- `Category` → single `Category`
- `Descriptions` → collection of `ProductDescription`
- `RelatedProducts` → collection of `Product` (many-to-many)

### `Categories`

| Property      | EDM type             | Nullable | Notes                |
|---------------|----------------------|----------|----------------------|
| `ID`          | Edm.Guid             | no       | Key                  |
| `Name`        | Edm.String (max 100) | no       | Unique, required     |
| `Description` | Edm.String (max 500) | yes      |                      |

Navigation: `Products` → collection of `Product`.

### `ProductDescriptions`

| Property      | EDM type              | Nullable | Notes                       |
|---------------|-----------------------|----------|-----------------------------|
| `ProductID`   | Edm.Guid              | no       | Key (part 1), FK to Product |
| `LanguageKey` | Edm.String (max 2)    | no       | Key (part 2), e.g. `EN`     |
| `Description` | Edm.String (max 500)  | no       | Searchable                  |
| `LongText`    | Edm.String (max 2000) | yes      | Searchable                  |
| `CustomName`  | Edm.String            | yes      |                             |

Navigation: `Product` → single `Product`. Addressable as
`/ProductDescriptions(ProductID=<guid>,LanguageKey='EN')`.

### `MediaItems`

Media entity (`HasStream=true`). Properties: `ID` (key), `Name`,
`ContentType`, `Size` (Edm.Int64, nullable), plus a binary media stream
addressable via `/MediaItems(<id>)/$value`.

### `ReadOnlyItems`

Properties: `ID` (key), `Name`. Carries entity-set capability annotations:
- `Org.OData.Capabilities.V1.InsertRestrictions` → `Insertable: false`
- `Org.OData.Capabilities.V1.UpdateRestrictions` → `Updatable: false`
- `Org.OData.Capabilities.V1.DeleteRestrictions` → `Deletable: false`

### `DecimalSamples`

Properties: `ID` (key), `Name`, `Amount` (**Edm.Decimal**, precision 38,
scale 18). Exists to isolate decimal behavior from `Product.Price` (Double).

### `Company` (singleton)

Addressable at `/Company`. Properties: `ID`, `Name`, `CEO`, `Founded`
(Edm.Int32), `HeadQuarter`, `Website`, `Logo` (Edm.Binary,
`image/svg+xml`), `Version` (**ETag**), `UpdatedAt` (Edm.DateTimeOffset).

---

## Complex types

**`Address`**: `Street` (max 100), `City` (max 50, searchable), `State`
(max 2), `PostalCode` (max 10), `Country` (max 50) — all Edm.String.

**`Dimensions`**: `Length`, `Width`, `Height` (Edm.Double, precision 10
scale 2), `Unit` (Edm.String max 10).

## Enum types

**`ProductStatus`** — a **flags** enum:

| Member         | Value |
|----------------|-------|
| `None`         | 0     |
| `InStock`      | 1     |
| `OnSale`       | 2     |
| `Discontinued` | 4     |
| `Featured`     | 8     |

## Derived types

**`SpecialProduct`** extends `Product`, adding `SpecialProperty` and
`SpecialFeature`. Used for type-cast / inheritance tests
(`/Products/<namespace>.SpecialProduct`).

---

## Operations (functions & actions)

The service must expose the following bound/unbound operations. Several are
**overloaded** (same name, different parameter sets) to exercise overload
resolution.

### Functions (composable, side-effect free)

| Function            | Binding         | Parameters                              |
|---------------------|-----------------|-----------------------------------------|
| `GetTopProducts`    | bound `Products` collection | `()`, `(count: Int64)`, `(count: Int64, category: String)` |
| `GetTotalPrice`     | bound `Products`            | `(taxRate: Double)`           |
| `GetProductStats`   | bound `Products` collection | `()`                          |
| `GetRelatedProducts`| bound `Products`            | `()`                          |
| `GetAveragePrice`   | bound `Products` collection | `()`                          |
| `FindProducts`      | unbound         | `(name: String, maxPrice: Double)`      |
| `Calculate`         | unbound         | `(value: Int64)`, `(a: Int64, b: Int64)`|
| `Convert`           | unbound         | `(input: String)`, `(number: Int64)`    |
| `CalculatePrice`    | bound `Products`            | `(discount: Double)`, `(discount: Double, tax: Double)` |
| `GetInfo`           | bound `Products`            | `(format: String)`            |

### Actions (may have side effects)

| Action             | Binding         | Parameters                  |
|--------------------|-----------------|-----------------------------|
| `ApplyDiscount`    | bound `Products`            | `(percentage: Double)`      |
| `IncreasePrice`    | bound `Products`            | `(amount: Double)`          |
| `Activate`         | bound `Products`            | `()`                        |
| `CalculateDiscount`| bound `Products`            | `(percentage: Double)`      |
| `MarkAllAsReviewed`| bound `Products` collection | `()`                        |
| `ResetAllPrices`   | unbound         | `()`                        |
| `ResetProducts`    | unbound         | `()`                        |
| `Process`          | unbound         | `(percentage: Double)`, `(percentage: Double, minPrice: Double)` |
| `Reseed`           | unbound         | `()` — resets DB to seed state |

> **`Reseed` is important for test isolation.** Several suites mutate data
> (create/update/delete). The reference server exposes an unbound `Reseed`
> action that restores the seed data so re-runs are deterministic.

---

## Seed data

Tests assert against specific rows, so the seed data must match:

**Categories** (3): `Electronics`, `Kitchen`, `Furniture`.

**Products** (7), with these `Name`s:
`Laptop` (a `SpecialProduct`, price 999.99), `Wireless Mouse` (29.99),
`Coffee Mug` (15.50), `Office Chair` (249.99), `Smartphone` (799.99),
`Premium Laptop Pro` (SpecialProduct, 1999.99),
`Gaming Mouse Ultra` (SpecialProduct, 149.99). The first product (`Laptop`)
carries full complex-type, geospatial, and duration sample values.

**ProductDescriptions**: multilingual rows for several products
(languages `EN`, `DE`, `FR`, `ES`), e.g. `Laptop` has `EN` + `DE`.

**Company** singleton: `TechStore Inc.`, CEO `Sarah Johnson`, founded 2010.

**DecimalSamples**: one row, `Amount = 123.450000000000000000`.

**ReadOnlyItems**: `Read-only item A`, `Read-only item B`.

See `cmd/complianceserver/entities/*.go` and `reseed.go` in the `go-odata`
repository for the exact, authoritative seed values.

---

## Roadmap

v1 hard-codes this reference model. Planned work to broaden applicability:

1. **Metadata-driven discovery** — read entity sets / key names / property
   types from the target's `$metadata` instead of assuming `Products` et al.
2. **Configurable fixtures** — a config file mapping the abstract roles the
   tests need (a "primary collection", a "composite-key set", a "singleton",
   a "media entity", a "flags enum") onto the target's actual names.
3. **Capability negotiation** — skip suites for features the target's
   `$metadata` / capability annotations declare unsupported, rather than
   failing them.

Until then, the simplest path for a non-`go-odata` service is to expose an
endpoint that reproduces the model above.
