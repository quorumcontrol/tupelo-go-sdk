package view

import (
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
)

// AggType represents the type of aggregation function used on a View.
type AggType int

// All available aggregation types.
const (
	AggTypeNone         AggType = iota // no aggregation; reserved for future use.
	AggTypeCount                       // the count aggregation, see Count.
	AggTypeSum                         // the sum aggregation, see Sum.
	AggTypeDistribution                // the distribution aggregation, see Distribution.
	AggTypeLastValue                   // the last value aggregation, see LastValue.
)

func (t AggType) String() string {
	return aggTypeName[t]
}

var aggTypeName = map[AggType]string{
	AggTypeNone:         "None",
	AggTypeCount:        "Count",
	AggTypeSum:          "Sum",
	AggTypeDistribution: "Distribution",
	AggTypeLastValue:    "LastValue",
}

type View struct {
	Name        string // Name of View. Must be unique. If unset, will default to the name of the Measure.
	Description string // Description is a human-readable description for this view.

	// TagKeys are the tag keys describing the grouping of this view.
	// A single Row will be produced for each combination of associated tag values.
	TagKeys []tag.Key

	// Measure is a stats.Measure to aggregate in this view.
	Measure stats.Measure

	// Aggregation is the aggregation function to apply to the set of Measurements.
	Aggregation *Aggregation
}

func Count() *Aggregation {
	return &Aggregation{}
}

func Register(views ...*View) error {
	return nil
}

type AggregationData interface {
}

type Aggregation struct {
	Type    AggType   // Type is the AggType of this Aggregation.
	Buckets []float64 // Buckets are the bucket endpoints if this Aggregation represents a distribution, see Distribution.

	newData func() AggregationData
}

func Distribution(bounds ...float64) *Aggregation {
	return &Aggregation{
		Type:    AggTypeDistribution,
		Buckets: bounds,
		newData: func() AggregationData {
			return nil
		},
	}
}
