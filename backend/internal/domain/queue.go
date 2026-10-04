package domain

import "time"

// Trend is how the truck queue changes, from the last two official snapshots.
type Trend struct {
	VehiclesPerHour float64   `json:"vehiclesPerHour"` // positive = growing
	Direction       string    `json:"direction"`       // "growing", "shrinking", "stable"
	From            time.Time `json:"from"`
	To              time.Time `json:"to"`
}

// HourOutlook is the expected wait for a truck arriving at the gate at At.
type HourOutlook struct {
	At          time.Time `json:"at"`
	Vehicles    int       `json:"vehicles"`
	WaitMinutes int       `json:"waitMinutes"`
	Level       Level     `json:"level"`
}
