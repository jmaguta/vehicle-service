package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/jmaguta/vehicle-service/internal/auth"
	mw "github.com/jmaguta/vehicle-service/internal/middleware"
)

// writeJSON wraps a successful payload in the { "data", "meta" } envelope (§5.3).
func writeJSON(w http.ResponseWriter, status int, v any) {
	writeJSONMeta(w, status, v, map[string]any{})
}

func writeJSONMeta(w http.ResponseWriter, status int, v any, meta map[string]any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"data": v, "meta": meta})
}

// writeError emits a flat { "error": msg } body — errors are not enveloped.
func writeError(w http.ResponseWriter, msg string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// workshopIDFromClaims returns the workshop_id from JWT claims in context.
// Returns "" for service-key calls (no claims); the repo treats "" as bypass.
func workshopIDFromClaims(r *http.Request) string {
	claims, ok := r.Context().Value(mw.ClaimsKey).(*auth.Claims)
	if !ok || claims == nil {
		return ""
	}
	return claims.WorkshopID
}

func stringValue(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func parseOptionalBool(raw string) (*bool, error) {
	if raw == "" {
		return nil, nil
	}
	value, err := strconv.ParseBool(strings.TrimSpace(raw))
	if err != nil {
		switch strings.ToLower(strings.TrimSpace(raw)) {
		case "yes":
			value = true
		case "no":
			value = false
		default:
			return nil, err
		}
	}
	return &value, nil
}
