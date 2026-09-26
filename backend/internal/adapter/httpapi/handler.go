// Package httpapi exposes the use cases as a JSON REST API.
package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/burakaydin/sinir-bekleme/backend/internal/domain"
	"github.com/burakaydin/sinir-bekleme/backend/internal/port"
	"github.com/burakaydin/sinir-bekleme/backend/internal/usecase"
)

type Handler struct {
	Status   *usecase.StatusService
	Trips    *usecase.TripService
	Geocoder port.Geocoder
	Log      *slog.Logger
}

// Routes:
//
//	GET /healthz
//	GET /v1/crossings                     all crossings with current estimates
//	GET /v1/crossings/{id}?hours=48       one crossing with raw history
//	POST /v1/trips/plan                   truck route with tachograph breaks and border waits
//	GET /v1/places?q=samsun               place search for origin/destination
func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("ok"))
	})
	mux.HandleFunc("GET /v1/crossings", h.listCrossings)
	mux.HandleFunc("GET /v1/crossings/{id}", h.getCrossing)
	mux.HandleFunc("POST /v1/trips/plan", h.planTrip)
	mux.HandleFunc("GET /v1/places", h.searchPlaces)
	return h.withLogging(withCORS(mux))
}

// withLogging records every request with its status and duration.
func (h *Handler) withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		h.Log.Info("http", "method", r.Method, "path", r.URL.Path, "status", rec.status,
			"duration", time.Since(start).Round(time.Millisecond), "remote", r.RemoteAddr)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func (h *Handler) listCrossings(w http.ResponseWriter, r *http.Request) {
	list, err := h.Status.List(r.Context())
	if err != nil {
		h.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"crossings": list})
}

func (h *Handler) getCrossing(w http.ResponseWriter, r *http.Request) {
	hours := 48
	if v := r.URL.Query().Get("hours"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 24*30 {
			writeJSON(w, http.StatusBadRequest, errorBody("hours must be between 1 and 720"))
			return
		}
		hours = n
	}
	since := time.Now().Add(-time.Duration(hours) * time.Hour)
	d, err := h.Status.Get(r.Context(), domain.CrossingID(r.PathValue("id")), since)
	if err != nil {
		h.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, d)
}

func (h *Handler) fail(w http.ResponseWriter, err error) {
	if errors.Is(err, domain.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, errorBody("not found"))
		return
	}
	h.Log.Error("request failed", "err", err)
	writeJSON(w, http.StatusInternalServerError, errorBody("internal error"))
}

func errorBody(msg string) map[string]string { return map[string]string{"error": msg} }

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
