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
// Returns "" for service-key calls (no claims).
func workshopIDFromClaims(r *http.Request) string {
	claims, ok := r.Context().Value(mw.ClaimsKey).(*auth.Claims)
	if !ok || claims == nil {
		return ""
	}
	return claims.WorkshopID
}

// resolveWorkshopID returns the workshop id to scope this request to,
// preferring JWT claims (user-authenticated calls) and falling back to the
// X-Workshop-Id header the BFF sends for service-key-only calls. ok=false
// means neither was present and the caller must reject the request rather
// than treat it as unscoped.
func resolveWorkshopID(r *http.Request) (id string, ok bool) {
	if wid := workshopIDFromClaims(r); wid != "" {
		return wid, true
	}
	if h := strings.TrimSpace(r.Header.Get("X-Workshop-Id")); h != "" {
		return h, true
	}
	return "", false
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
