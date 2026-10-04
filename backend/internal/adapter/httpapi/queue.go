package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/burakaydin/sinir-bekleme/backend/internal/domain"
	"github.com/burakaydin/sinir-bekleme/backend/internal/usecase"
)

func (h *Handler) getQueue(w http.ResponseWriter, r *http.Request) {
	d, err := h.Queue.Detail(r.Context(), domain.CrossingID(r.PathValue("id")))
	if err != nil {
		h.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, d)
}

type reportRequest struct {
	Direction     domain.Direction  `json:"direction"`
	Kind          domain.ReportKind `json:"kind"`
	At            *time.Time        `json:"at"`
	VehiclesAhead *int              `json:"vehiclesAhead"`
	QueueKm       *float64          `json:"queueKm"`
	WaitMinutes   *int              `json:"waitMinutes"`
	Note          string            `json:"note"`
}

func (h *Handler) postReport(w http.ResponseWriter, r *http.Request) {
	var req reportRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<12)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorBody("invalid JSON body"))
		return
	}
	rep := domain.DriverReport{
		CrossingID: domain.CrossingID(r.PathValue("id")), Direction: req.Direction, Kind: req.Kind,
		VehiclesAhead: req.VehiclesAhead, QueueKm: req.QueueKm, WaitMinutes: req.WaitMinutes, Note: req.Note,
	}
	if req.At != nil {
		rep.At = *req.At
	}
	saved, err := h.Queue.Report(r.Context(), rep)
	if errors.Is(err, usecase.ErrInvalidReport) {
		writeJSON(w, http.StatusBadRequest, errorBody(err.Error()))
		return
	}
	if err != nil {
		h.fail(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, saved)
}
