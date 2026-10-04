package domain

import "time"

// ReportKind is what a driver tells about a crossing.
type ReportKind string

const (
	// ReportInQueue: the driver is in the queue now.
	ReportInQueue ReportKind = "in_queue"
	// ReportPassed: the driver has just crossed; WaitMinutes is the measured wait.
	ReportPassed ReportKind = "passed"
)

// DriverReport is a first-hand observation sent from the app.
type DriverReport struct {
	ID         string     `json:"id"`
	CrossingID CrossingID `json:"crossingId"`
	Direction  Direction  `json:"direction"`
	Kind       ReportKind `json:"kind"`
	At         time.Time  `json:"at"`

	VehiclesAhead *int     `json:"vehiclesAhead,omitempty"` // in_queue
	QueueKm       *float64 `json:"queueKm,omitempty"`       // in_queue: distance to the gate
	WaitMinutes   *int     `json:"waitMinutes,omitempty"`   // passed: total wait
	Note          string   `json:"note,omitempty"`
}

// Limits keep obviously wrong reports out.
const (
	MaxReportVehicles = 5000
	MaxReportQueueKm  = 80
	MaxReportWaitMin  = 7 * 24 * 60
	MaxReportNoteLen  = 280
)
