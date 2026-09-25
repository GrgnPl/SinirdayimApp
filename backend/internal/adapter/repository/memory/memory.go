// Package memory is an in-process SnapshotRepository, used for development
// and tests until the PostgreSQL adapter lands.
package memory

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/burakaydin/sinir-bekleme/backend/internal/domain"
)

type key struct {
	crossing   domain.CrossingID
	direction  domain.Direction
	source     string
	observedAt int64
}

type Repository struct {
	mu         sync.RWMutex
	seen       map[key]struct{}
	byCrossing map[domain.CrossingID][]domain.Snapshot // sorted by ObservedAt ascending
}

func New() *Repository {
	return &Repository{
		seen:       map[key]struct{}{},
		byCrossing: map[domain.CrossingID][]domain.Snapshot{},
	}
}

func (r *Repository) Save(_ context.Context, snaps []domain.Snapshot) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	touched := map[domain.CrossingID]bool{}
	for _, s := range snaps {
		k := key{s.CrossingID, s.Direction, s.Source, s.ObservedAt.UnixNano()}
		if _, dup := r.seen[k]; dup {
			continue
		}
		r.seen[k] = struct{}{}
		r.byCrossing[s.CrossingID] = append(r.byCrossing[s.CrossingID], s)
		touched[s.CrossingID] = true
	}
	for id := range touched {
		list := r.byCrossing[id]
		sort.SliceStable(list, func(i, j int) bool { return list[i].ObservedAt.Before(list[j].ObservedAt) })
	}
	return nil
}

func (r *Repository) Latest(_ context.Context, id domain.CrossingID) ([]domain.Snapshot, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	type dk struct {
		d domain.Direction
		s string
	}
	newest := map[dk]domain.Snapshot{}
	for _, s := range r.byCrossing[id] {
		newest[dk{s.Direction, s.Source}] = s // ascending order, so last wins
	}
	out := make([]domain.Snapshot, 0, len(newest))
	for _, s := range newest {
		out = append(out, s)
	}
	return out, nil
}

func (r *Repository) History(_ context.Context, id domain.CrossingID, since time.Time) ([]domain.Snapshot, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []domain.Snapshot
	for _, s := range r.byCrossing[id] {
		if !s.ObservedAt.Before(since) {
			out = append(out, s)
		}
	}
	return out, nil
}
