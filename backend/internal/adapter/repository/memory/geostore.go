package memory

import (
	"context"
	"sync"

	"github.com/burakaydin/sinir-bekleme/backend/internal/domain"
)

// GeoStore is an in-process port.GeoStore.
type GeoStore[T any] struct {
	mu       sync.RWMutex
	bySource map[string][]T
	location func(T) domain.GeoPoint
}

func NewGeoStore[T any](location func(T) domain.GeoPoint) *GeoStore[T] {
	return &GeoStore[T]{bySource: map[string][]T{}, location: location}
}

func NewRestAreas() *GeoStore[domain.RestArea] {
	return NewGeoStore(func(a domain.RestArea) domain.GeoPoint { return a.Location })
}

func NewBorderPoints() *GeoStore[domain.BorderPoint] {
	return NewGeoStore(func(b domain.BorderPoint) domain.GeoPoint { return b.Location })
}

func (g *GeoStore[T]) Replace(_ context.Context, sourceID string, items []T) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.bySource[sourceID] = append([]T(nil), items...)
	return nil
}

func (g *GeoStore[T]) InBounds(_ context.Context, sw, ne domain.GeoPoint) ([]T, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	var out []T
	for _, items := range g.bySource {
		for _, it := range items {
			p := g.location(it)
			if p.Lat >= sw.Lat && p.Lat <= ne.Lat && p.Lng >= sw.Lng && p.Lng <= ne.Lng {
				out = append(out, it)
			}
		}
	}
	return out, nil
}
