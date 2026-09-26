package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/burakaydin/sinir-bekleme/backend/internal/domain"
	"github.com/burakaydin/sinir-bekleme/backend/internal/tacho"
	"github.com/burakaydin/sinir-bekleme/backend/internal/usecase"
)

type planRequest struct {
	Origin      *domain.GeoPoint `json:"origin"`
	Destination *domain.GeoPoint `json:"destination"`
	DepartAt    *time.Time       `json:"departAt"`
	Driver      struct {
		ContinuousDrivingMin int        `json:"continuousDrivingMin"`
		DailyDrivingMin      int        `json:"dailyDrivingMin"`
		DutyStartedAt        *time.Time `json:"dutyStartedAt"`
		ExtendedDaysLeft     *int       `json:"extendedDaysLeft"`
	} `json:"driver"`
}

func (h *Handler) planTrip(w http.ResponseWriter, r *http.Request) {
	var req planRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorBody("invalid JSON body"))
		return
	}
	if req.Origin == nil || req.Destination == nil {
		writeJSON(w, http.StatusBadRequest, errorBody("origin and destination are required"))
		return
	}

	in := usecase.TripRequest{
		Origin:      *req.Origin,
		Destination: *req.Destination,
		DepartAt:    time.Now().Truncate(time.Minute),
		Driver: tacho.DriverState{
			ContinuousDriving: time.Duration(req.Driver.ContinuousDrivingMin) * time.Minute,
			DailyDriving:      time.Duration(req.Driver.DailyDrivingMin) * time.Minute,
			ExtendedDaysLeft:  2,
		},
	}
	if req.DepartAt != nil {
		in.DepartAt = *req.DepartAt
	}
	if req.Driver.DutyStartedAt != nil {
		in.Driver.DutyStartedAt = *req.Driver.DutyStartedAt
	} else {
		// Without an explicit start, assume the duty period began when
		// today's driving began: the most conservative guess we can make.
		in.Driver.DutyStartedAt = in.DepartAt.Add(-in.Driver.DailyDriving)
	}
	if req.Driver.ExtendedDaysLeft != nil {
		in.Driver.ExtendedDaysLeft = *req.Driver.ExtendedDaysLeft
	}

	plan, err := h.Trips.Plan(r.Context(), in)
	switch {
	case errors.Is(err, tacho.ErrInvalidState):
		writeJSON(w, http.StatusBadRequest, errorBody("driver state is outside tachograph limits"))
	case errors.Is(err, domain.ErrNoRoute):
		writeJSON(w, http.StatusUnprocessableEntity, errorBody("no truck route found between these points"))
	case err != nil:
		h.fail(w, err)
	default:
		writeJSON(w, http.StatusOK, plan)
	}
}

func (h *Handler) searchPlaces(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if len([]rune(q)) < 2 {
		writeJSON(w, http.StatusOK, map[string]any{"places": []domain.Place{}})
		return
	}
	limit := 6
	if v, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && v > 0 && v <= 20 {
		limit = v
	}
	places, err := h.Geocoder.Search(r.Context(), q, limit)
	if err != nil {
		h.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"places": places})
}
