package framework

// ConformanceLevel is the historical API name for a suite coverage band.
// It must not be interpreted as a formal OASIS certification by itself.
type ConformanceLevel int

const (
	LevelUnspecified  ConformanceLevel = iota
	LevelMinimal                       // basic read, metadata, core HTTP
	LevelIntermediate                  // filter, sort, paging, count, CRUD
	LevelAdvanced                      // expand, search, batch, async, ETags
)

func (l ConformanceLevel) String() string {
	switch l {
	case LevelMinimal:
		return "Minimal"
	case LevelIntermediate:
		return "Intermediate"
	case LevelAdvanced:
		return "Advanced"
	default:
		return "Unspecified"
	}
}
