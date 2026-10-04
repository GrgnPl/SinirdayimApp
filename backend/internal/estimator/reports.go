package estimator

import (
	"sort"
	"time"

	"github.com/burakaydin/sinir-bekleme/backend/internal/domain"
)

const (
	// Reports older than these are ignored for the current state.
	passedReportWindow  = 3 * time.Hour
	inQueueReportWindow = 90 * time.Minute
)

// WithReports refines an official estimate with recent driver reports for
// the same crossing and direction. Measured waits ("passed") win; queue
// reports ("in_queue") give the current vehicle count. Medians keep a
// single wrong report from moving the result.
func WithReports(est domain.WaitEstimate, c domain.Crossing, reports []domain.DriverReport, dailyThroughput *int, now time.Time) domain.WaitEstimate {
	var waits, vehicles []int
	var newest time.Time
	for _, r := range reports {
		if r.CrossingID != c.ID || r.Direction != est.Direction {
			continue
		}
		age := now.Sub(r.At)
		switch {
		case r.Kind == domain.ReportPassed && r.WaitMinutes != nil && age <= passedReportWindow:
			waits = append(waits, *r.WaitMinutes)
		case r.Kind == domain.ReportInQueue && age <= inQueueReportWindow:
			if v := reportVehicles(c, r); v != nil {
				vehicles = append(vehicles, *v)
			}
		default:
			continue
		}
		if r.At.After(newest) {
			newest = r.At
		}
	}
	if len(waits) == 0 && len(vehicles) == 0 {
		return est
	}

	out := est
	out.BasedOn = appendUnique(append([]string(nil), est.BasedOn...), "drivers")
	out.DataAt = newest
	if len(vehicles) > 0 {
		out.Vehicles = domain.Ptr(median(vehicles))
	}
	switch {
	case len(waits) > 0:
		out.WaitMinutes = domain.Ptr(median(waits))
		out.Method = "driver_reports"
		out.Confidence = confidenceFor(len(waits))
	case dailyThroughput != nil && *dailyThroughput > 0:
		perHour := float64(*dailyThroughput) / 24
		out.WaitMinutes = domain.Ptr(int(float64(*out.Vehicles) / perHour * 60))
		out.Method = "driver_queue/throughput"
		out.Confidence = confidenceFor(len(vehicles))
	default:
		out.WaitMinutes = nil
		out.Method = "driver_queue"
		out.Confidence = domain.ConfidenceLow
	}
	out.Level = level(out)
	return out
}

func reportVehicles(c domain.Crossing, r domain.DriverReport) *int {
	switch {
	case r.VehiclesAhead != nil:
		return r.VehiclesAhead
	case r.QueueKm != nil:
		return domain.Ptr(int(*r.QueueKm * trucksPerKmPerLane * float64(max(c.Lanes, 1))))
	}
	return nil
}

func confidenceFor(n int) domain.Confidence {
	if n >= 2 {
		return domain.ConfidenceHigh
	}
	return domain.ConfidenceMedium
}

func median(v []int) int {
	s := append([]int(nil), v...)
	sort.Ints(s)
	m := len(s) / 2
	if len(s)%2 == 1 {
		return s[m]
	}
	return (s[m-1] + s[m]) / 2
}

func appendUnique(list []string, v string) []string {
	for _, x := range list {
		if x == v {
			return list
		}
	}
	return append(list, v)
}
