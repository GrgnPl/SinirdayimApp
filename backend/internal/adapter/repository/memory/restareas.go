package memory

import (
	"context"
	"sync"

	"github.com/burakaydin/sinir-bekleme/backend/internal/domain"
)

// RestAreas is an in-process port.RestAreaStore.
type RestAreas struct {
	mu       sync.RWMutex
	bySource map[string][]domain.RestArea
}

func NewRestAreas() *RestAreas {
	return &RestAreas{bySource: map[string][]domain.RestArea{}}
}

func (r *RestAreas) Replace(_ context.Context, sourceID string, areas []domain.RestArea) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.bySource[sourceID] = append([]domain.RestArea(nil), areas...)
	return nil
}

func (r *RestAreas) InBounds(_ context.Context, sw, ne domain.GeoPoint) ([]domain.RestArea, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []domain.RestArea
	for _, areas := range r.bySource {
		for _, a := range areas {
			if a.Location.Lat >= sw.Lat && a.Location.Lat <= ne.Lat &&
				a.Location.Lng >= sw.Lng && a.Location.Lng <= ne.Lng {
				out = append(out, a)
			}
		}
	}
	return out, nil
}
