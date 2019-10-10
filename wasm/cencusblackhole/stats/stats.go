package stats

// Units are encoded according to the case-sensitive abbreviations from the
// Unified Code for Units of Measure: http://unitsofmeasure.org/ucum.html
const (
	UnitNone          = "1" // Deprecated: Use UnitDimensionless.
	UnitDimensionless = "1"
	UnitBytes         = "By"
	UnitMilliseconds  = "ms"
)

type View struct {
	// Name        string // Name of View. Must be unique. If unset, will default to the name of the Measure.
	// Description string // Description is a human-readable description for this view.

	// // TagKeys are the tag keys describing the grouping of this view.
	// // A single Row will be produced for each combination of associated tag values.
	// TagKeys []tag.Key

	// // Measure is a stats.Measure to aggregate in this view.
	// Measure stats.Measure

	// // Aggregation is the aggregation function to apply to the set of Measurements.
	// Aggregation *Aggregation
}

func Register(views ...*View) error {
	return nil
}
