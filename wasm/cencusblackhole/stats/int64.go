package stats

// Int64Measure is a measure for int64 values.
type Int64Measure struct {
}

type Measure interface {
	// Name returns the name of this measure.
	//
	// Measure names are globally unique (among all libraries linked into your program).
	// We recommend prefixing the measure name with a domain name relevant to your
	// project or application.
	//
	// Measure names are never sent over the wire or exported to backends.
	// They are only used to create Views.
	Name() string

	// Description returns the human-readable description of this measure.
	Description() string

	// Unit returns the units for the values this measure takes on.
	//
	// Units are encoded according to the case-sensitive abbreviations from the
	// Unified Code for Units of Measure: http://unitsofmeasure.org/ucum.html
	Unit() string
}

type Measurement struct {
	v    float64
	m    Measure
	desc string
}

// Value returns the value of the Measurement as a float64.
func (m Measurement) Value() float64 {
	return m.v
}

// Measure returns the Measure from which this Measurement was created.
func (m Measurement) Measure() Measure {
	return m.m
}

// M creates a new int64 measurement.
// Use Record to record measurements.
func (m *Int64Measure) M(v int64) Measurement {
	return Measurement{
		m:    m,
		desc: "",
		v:    float64(v),
	}
}

// Int64 creates a new measure for int64 values.
//
// See the documentation for interface Measure for more guidance on the
// parameters of this function.
func Int64(name, description, unit string) *Int64Measure {
	return &Int64Measure{}
}

// Name returns the name of the measure.
func (m *Int64Measure) Name() string {
	return ""
}

// Description returns the description of the measure.
func (m *Int64Measure) Description() string {
	return ""
}

// Unit returns the unit of the measure.
func (m *Int64Measure) Unit() string {
	return ""
}
