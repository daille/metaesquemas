// Package httpapi expone la API REST/JSON de lectura del catálogo y la ingesta administrativa.
package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/org/metaesquemas/internal/domain"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, log *slog.Logger, err error) {
	status, code, msg := http.StatusInternalServerError, "internal", "Error interno del servidor."
	switch {
	case errors.Is(err, domain.ErrNotFound):
		status, code, msg = http.StatusNotFound, "not_found", "Recurso no encontrado."
	case errors.Is(err, domain.ErrInvalid):
		status, code, msg = http.StatusBadRequest, "invalid_request", err.Error()
	}
	if status >= 500 {
		log.Error("request failed", "err", err)
	}
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": msg}})
}
