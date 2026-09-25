package domain

import "time"

// Snapshot is one observation of a crossing's queue, as reported by a source.
// Every metric is optional because sources publish different subsets.
type Snapshot struct {
	CrossingID CrossingID `json:"crossingId"`
	Direction  Direction  `json:"direction"`
	Source     string     `json:"source"` // provider id, e.g. "und"
	ObservedAt time.Time  `json:"observedAt"`

	QueueKm         *float64 `json:"queueKm,omitempty"`         // length of the truck queue
	QueueVehicles   *int     `json:"queueVehicles,omitempty"`   // trucks waiting in the queue
	ParkedVehicles  *int     `json:"parkedVehicles,omitempty"`  // trucks waiting in the TIR park
	DailyThroughput *int     `json:"dailyThroughput,omitempty"` // trucks processed in the last 24h
	WaitMinutes     *int     `json:"waitMinutes,omitempty"`     // wait time reported directly by the source
}

// Confidence expresses how much an estimate can be trusted.
type Confidence string

const (
	ConfidenceHigh   Confidence = "high"
	ConfidenceMedium Confidence = "medium"
	ConfidenceLow    Confidence = "low"
)

// Level is a coarse traffic level for UI coloring.
type Level string

const (
	LevelLow     Level = "low"
	LevelMedium  Level = "medium"
	LevelHigh    Level = "high"
	LevelUnknown Level = "unknown"
)

// WaitEstimate is the derived answer the user actually cares about.
type WaitEstimate struct {
	CrossingID  CrossingID `json:"crossingId"`
	Direction   Direction  `json:"direction"`
	Vehicles    *int       `json:"vehicles,omitempty"`
	WaitMinutes *int       `json:"waitMinutes,omitempty"`
	Level       Level      `json:"level"`
	Confidence  Confidence `json:"confidence"`
	Method      string     `json:"method"`  // how the estimate was produced
	BasedOn     []string   `json:"basedOn"` // source ids used
	DataAt      time.Time  `json:"dataAt"`  // observation time of the newest input
}

// Ptr is a small helper for building optional fields.
func Ptr[T any](v T) *T { return &v }
