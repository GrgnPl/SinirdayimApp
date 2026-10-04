package memory

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/GrgnPl/SinirdayimApp/backend/internal/domain"
)

// Reports is an in-process port.ReportStore.
type Reports struct {
	mu   sync.RWMutex
	byID map[domain.CrossingID][]domain.DriverReport
}

func NewReports() *Reports {
	return &Reports{byID: map[domain.CrossingID][]domain.DriverReport{}}
}

func (r *Reports) Add(_ context.Context, rep domain.DriverReport) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byID[rep.CrossingID] = append(r.byID[rep.CrossingID], rep)
	return nil
}

func (r *Reports) Since(_ context.Context, id domain.CrossingID, t time.Time) ([]domain.DriverReport, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []domain.DriverReport
	for _, rep := range r.byID[id] {
		if !rep.At.Before(t) {
			out = append(out, rep)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].At.After(out[j].At) })
	return out, nil
}
