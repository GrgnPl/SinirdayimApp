package estimator

import (
	"math"
	"sort"
	"time"

	"github.com/burakaydin/sinir-bekleme/backend/internal/domain"
)

const (
	// trendHorizon limits how far a measured trend is extrapolated; beyond
	// it the queue is assumed to stay as it is.
	trendHorizon = 6 * time.Hour
	// trendMaxAge drops trends computed from old snapshot pairs.
	trendMaxAge = 36 * time.Hour
)

// TrendOf compares the newest two snapshots of one source and direction that
// both give a vehicle count. Nil when there are not two usable snapshots.
func TrendOf(c domain.Crossing, history []domain.Snapshot, now time.Time) *domain.Trend {
	var pts []domain.Snapshot
	for _, s := range history {
		if vehicles(c, s) != nil {
			pts = append(pts, s)
		}
	}
	if len(pts) < 2 {
		return nil
	}
	sort.Slice(pts, func(i, j int) bool { return pts[i].ObservedAt.Before(pts[j].ObservedAt) })
	a, b := pts[len(pts)-2], pts[len(pts)-1]
	span := b.ObservedAt.Sub(a.ObservedAt)
	if span <= 0 || now.Sub(b.ObservedAt) > trendMaxAge {
		return nil
	}
	rate := float64(*vehicles(c, b)-*vehicles(c, a)) / span.Hours()
	t := &domain.Trend{VehiclesPerHour: math.Round(rate*10) / 10, From: a.ObservedAt, To: b.ObservedAt, Direction: "stable"}
	switch {
	case rate > 2:
		t.Direction = "growing"
	case rate < -2:
		t.Direction = "shrinking"
	}
	return t
}

// Outlook projects the wait hour by hour from now: the queue moves by the
// trend for up to trendHorizon, and is served at the daily throughput.
// Nil when either the vehicle count or the throughput is unknown.
func Outlook(est domain.WaitEstimate, throughput *int, trend *domain.Trend, now time.Time, hours int) []domain.HourOutlook {
	if est.Vehicles == nil || throughput == nil || *throughput <= 0 {
		return nil
	}
	perHour := float64(*throughput) / 24
	rate := 0.0
	if trend != nil {
		// A queue cannot shrink faster than the gate processes trucks.
		rate = math.Max(trend.VehiclesPerHour, -perHour)
	}
	base := float64(*est.Vehicles)
	// Shift the projection to "now" if the estimate is older.
	if !est.DataAt.IsZero() && now.After(est.DataAt) {
		base += rate * math.Min(now.Sub(est.DataAt).Hours(), trendHorizon.Hours())
	}
	start := now.Truncate(time.Hour)
	out := make([]domain.HourOutlook, 0, hours)
	for h := 0; h < hours; h++ {
		at := start.Add(time.Duration(h) * time.Hour)
		dt := math.Min(math.Max(at.Sub(now).Hours(), 0), trendHorizon.Hours())
		v := math.Max(base+rate*dt, 0)
		wait := int(v / perHour * 60)
		o := domain.HourOutlook{At: at, Vehicles: int(v), WaitMinutes: wait}
		o.Level = level(domain.WaitEstimate{WaitMinutes: &wait})
		out = append(out, o)
	}
	return out
}

// Methods implementing port.Estimator.

func (q *Queue) WithReports(est domain.WaitEstimate, c domain.Crossing, reports []domain.DriverReport, dailyThroughput *int) domain.WaitEstimate {
	return WithReports(est, c, reports, dailyThroughput, q.Now())
}

func (q *Queue) Trend(c domain.Crossing, history []domain.Snapshot) *domain.Trend {
	return TrendOf(c, history, q.Now())
}

func (q *Queue) Outlook(est domain.WaitEstimate, dailyThroughput *int, trend *domain.Trend, hours int) []domain.HourOutlook {
	return Outlook(est, dailyThroughput, trend, q.Now(), hours)
}
