// Package estimator converts raw queue observations into wait-time estimates.
package estimator

import (
	"sort"
	"time"

	"github.com/burakaydin/sinir-bekleme/backend/internal/domain"
)

const (
	// trucksPerKmPerLane assumes ~18 m per stopped truck (16.5 m rig + gap).
	trucksPerKmPerLane = 55
	// staleAfter marks data older than this as low confidence.
	staleAfter = 12 * time.Hour
)

// Queue estimates wait as (trucks waiting) / (trucks processed per hour).
// It is the baseline model; a learned model can replace it behind port.Estimator.
type Queue struct {
	Now func() time.Time
}

func NewQueue() *Queue { return &Queue{Now: time.Now} }

func (q *Queue) Estimate(c domain.Crossing, dir domain.Direction, latest []domain.Snapshot) domain.WaitEstimate {
	est := domain.WaitEstimate{
		CrossingID: c.ID,
		Direction:  dir,
		Level:      domain.LevelUnknown,
		Confidence: domain.ConfidenceLow,
		Method:     "none",
		BasedOn:    []string{},
	}

	snaps := make([]domain.Snapshot, 0, len(latest))
	for _, s := range latest {
		if s.Direction == dir {
			snaps = append(snaps, s)
		}
	}
	if len(snaps) == 0 {
		return est
	}
	sort.Slice(snaps, func(i, j int) bool { return snaps[i].ObservedAt.After(snaps[j].ObservedAt) })

	// A source that reports wait time directly wins.
	for _, s := range snaps {
		if s.WaitMinutes != nil {
			est.WaitMinutes = s.WaitMinutes
			est.Vehicles = vehicles(c, s)
			est.Method = "reported"
			est.Confidence = domain.ConfidenceHigh
			est.BasedOn = []string{s.Source}
			est.DataAt = s.ObservedAt
			return q.finish(est)
		}
	}

	s := snaps[0]
	est.BasedOn = []string{s.Source}
	est.DataAt = s.ObservedAt
	est.Vehicles = vehicles(c, s)
	if est.Vehicles == nil {
		return q.finish(est)
	}

	if s.DailyThroughput != nil && *s.DailyThroughput > 0 {
		perHour := float64(*s.DailyThroughput) / 24
		est.WaitMinutes = domain.Ptr(int(float64(*est.Vehicles) / perHour * 60))
		est.Method = "queue/throughput"
		est.Confidence = domain.ConfidenceMedium
	} else {
		est.Method = "queue-only"
	}
	return q.finish(est)
}

func (q *Queue) finish(est domain.WaitEstimate) domain.WaitEstimate {
	if q.Now().Sub(est.DataAt) > staleAfter {
		est.Confidence = domain.ConfidenceLow
	}
	est.Level = level(est)
	return est
}

// vehicles returns trucks waiting (queue + TIR park), or nil if unknown.
func vehicles(c domain.Crossing, s domain.Snapshot) *int {
	var n int
	known := false
	switch {
	case s.QueueVehicles != nil:
		n += *s.QueueVehicles
		known = true
	case s.QueueKm != nil:
		lanes := max(c.Lanes, 1)
		n += int(*s.QueueKm * trucksPerKmPerLane * float64(lanes))
		known = true
	}
	if s.ParkedVehicles != nil {
		n += *s.ParkedVehicles
		known = true
	}
	if !known {
		return nil
	}
	return &n
}

func level(est domain.WaitEstimate) domain.Level {
	if est.WaitMinutes != nil {
		switch h := *est.WaitMinutes / 60; {
		case h < 2:
			return domain.LevelLow
		case h < 8:
			return domain.LevelMedium
		default:
			return domain.LevelHigh
		}
	}
	if est.Vehicles != nil {
		switch v := *est.Vehicles; {
		case v < 100:
			return domain.LevelLow
		case v < 500:
			return domain.LevelMedium
		default:
			return domain.LevelHigh
		}
	}
	return domain.LevelUnknown
}
