package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/GrgnPl/SinirdayimApp/backend/internal/domain"
	"github.com/GrgnPl/SinirdayimApp/backend/internal/tacho"
	"github.com/GrgnPl/SinirdayimApp/backend/internal/usecase"
)

type planRequest struct {
	Origin      *domain.GeoPoint `json:"origin"`
	Destination *domain.GeoPoint `json:"destination"`
	DepartAt    *time.Time       `json:"departAt"`
	// StateAt is when the driver state was entered; defaults to departAt.
	StateAt *time.Time `json:"stateAt"`
	Driver  struct {
		ContinuousDrivingMin int        `json:"continuousDrivingMin"`
		SplitBreakTaken      bool       `json:"splitBreakTaken"`
		DailyDrivingMin      int        `json:"dailyDrivingMin"`
		DutyStartedAt        *time.Time `json:"dutyStartedAt"`
		ExtendedDaysLeft     *int       `json:"extendedDaysLeft"`
		ReducedRestsLeft     int        `json:"reducedRestsLeft"`
		WeeklyDrivingMin     int        `json:"weeklyDrivingMin"`
		PrevWeekDrivingMin   int        `json:"prevWeekDrivingMin"`
		LastWeeklyRestEnd    *time.Time `json:"lastWeeklyRestEnd"`
	} `json:"driver"`
	// Appointment is a booked slot, e.g. RSS at Kapıkule.
	Appointment *struct {
		CrossingID string    `json:"crossingId"`
		At         time.Time `json:"at"`
	} `json:"appointment"`
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
			SplitBreakTaken:   req.Driver.SplitBreakTaken,
			DailyDriving:      time.Duration(req.Driver.DailyDrivingMin) * time.Minute,
			ExtendedDaysLeft:  2,
			ReducedRestsLeft:  req.Driver.ReducedRestsLeft,
			WeeklyDriving:     time.Duration(req.Driver.WeeklyDrivingMin) * time.Minute,
			PrevWeekDriving:   time.Duration(req.Driver.PrevWeekDrivingMin) * time.Minute,
		},
	}
	if req.DepartAt != nil {
		in.DepartAt = *req.DepartAt
	}
	stateAt := in.DepartAt
	if req.StateAt != nil && !req.StateAt.After(in.DepartAt) {
		stateAt = *req.StateAt
		in.StateAt = stateAt
	}
	if req.Driver.DutyStartedAt != nil {
		in.Driver.DutyStartedAt = *req.Driver.DutyStartedAt
	} else {
		// Without an explicit start, assume the duty period began when
		// today's driving began: the most conservative guess we can make.
		in.Driver.DutyStartedAt = stateAt.Add(-in.Driver.DailyDriving)
	}
	if a := req.Appointment; a != nil {
		if a.CrossingID == "" || a.At.IsZero() {
			writeJSON(w, http.StatusBadRequest, errorBody("appointment needs crossingId and at"))
			return
		}
		in.Appointment = &usecase.Appointment{CrossingID: domain.CrossingID(a.CrossingID), At: a.At}
	}
	if req.Driver.LastWeeklyRestEnd != nil {
		in.Driver.LastWeeklyRestEnd = *req.Driver.LastWeeklyRestEnd
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
	case errors.Is(err, usecase.ErrAppointmentNotOnRoute):
		writeJSON(w, http.StatusUnprocessableEntity, errorBody("the appointment crossing is not on a route between these points"))
	case errors.Is(err, domain.ErrNotFound):
		writeJSON(w, http.StatusNotFound, errorBody("unknown crossing"))
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
