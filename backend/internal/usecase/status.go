// Package usecase contains the application logic. It depends only on domain
// types and port interfaces.
package usecase

import (
	"context"
	"time"

	"github.com/burakaydin/sinir-bekleme/backend/internal/domain"
	"github.com/burakaydin/sinir-bekleme/backend/internal/port"
)

// CrossingStatus is a crossing together with its current estimates.
type CrossingStatus struct {
	domain.Crossing
	Export domain.WaitEstimate `json:"export"`
	Import domain.WaitEstimate `json:"import"`
}

// CrossingDetail adds recent raw observations for charts.
type CrossingDetail struct {
	CrossingStatus
	History []domain.Snapshot `json:"history"`
}

type StatusService struct {
	Catalog   port.CrossingCatalog
	Repo      port.SnapshotRepository
	Estimator port.Estimator
}

func (s *StatusService) List(ctx context.Context) ([]CrossingStatus, error) {
	crossings, err := s.Catalog.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]CrossingStatus, 0, len(crossings))
	for _, c := range crossings {
		st, err := s.status(ctx, c)
		if err != nil {
			return nil, err
		}
		out = append(out, st)
	}
	return out, nil
}

func (s *StatusService) Get(ctx context.Context, id domain.CrossingID, historySince time.Time) (CrossingDetail, error) {
	c, err := s.Catalog.Get(ctx, id)
	if err != nil {
		return CrossingDetail{}, err
	}
	st, err := s.status(ctx, c)
	if err != nil {
		return CrossingDetail{}, err
	}
	hist, err := s.Repo.History(ctx, id, historySince)
	if err != nil {
		return CrossingDetail{}, err
	}
	if hist == nil {
		hist = []domain.Snapshot{}
	}
	return CrossingDetail{CrossingStatus: st, History: hist}, nil
}

// Estimate returns the current wait estimate for one crossing and direction.
func (s *StatusService) Estimate(ctx context.Context, c domain.Crossing, dir domain.Direction) (domain.WaitEstimate, error) {
	latest, err := s.Repo.Latest(ctx, c.ID)
	if err != nil {
		return domain.WaitEstimate{}, err
	}
	return s.Estimator.Estimate(c, dir, latest), nil
}

func (s *StatusService) status(ctx context.Context, c domain.Crossing) (CrossingStatus, error) {
	latest, err := s.Repo.Latest(ctx, c.ID)
	if err != nil {
		return CrossingStatus{}, err
	}
	return CrossingStatus{
		Crossing: c,
		Export:   s.Estimator.Estimate(c, domain.DirectionExport, latest),
		Import:   s.Estimator.Estimate(c, domain.DirectionImport, latest),
	}, nil
}
