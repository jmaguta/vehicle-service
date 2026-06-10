package openapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestHandler_ServesValid31Document(t *testing.T) {
	r := chi.NewRouter()
	r.Get("/checklists", func(w http.ResponseWriter, _ *http.Request) {})
	r.Post("/checklists", func(w http.ResponseWriter, _ *http.Request) {})
	r.Get("/sites/{id:[0-9a-f-]+}", func(w http.ResponseWriter, _ *http.Request) {})
	r.Get("/openapi.json", Handler(r, "vehicle-service", "1.2.3"))

	req := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", ct)
	}

	var doc struct {
		OpenAPI string `json:"openapi"`
		Info    struct {
			Title   string `json:"title"`
			Version string `json:"version"`
		} `json:"info"`
		Paths map[string]map[string]any `json:"paths"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &doc); err != nil {
		t.Fatalf("body is not valid JSON: %v", err)
	}

	if doc.OpenAPI != "3.1.0" {
		t.Errorf("openapi = %q, want 3.1.0", doc.OpenAPI)
	}
	if doc.Info.Title != "vehicle-service" || doc.Info.Version != "1.2.3" {
		t.Errorf("info = %+v, want title=vehicle-service version=1.2.3", doc.Info)
	}
	if len(doc.Paths) == 0 {
		t.Fatal("paths is empty, want the walked routes")
	}
	// Regex constraint on the path param must be stripped to bare {id}.
	if _, ok := doc.Paths["/sites/{id}"]; !ok {
		t.Errorf("missing normalised path /sites/{id}; got paths %v", keys(doc.Paths))
	}
	// Both methods on /checklists should be present as separate operations.
	if ops := doc.Paths["/checklists"]; ops["get"] == nil || ops["post"] == nil {
		t.Errorf("/checklists ops = %v, want get+post", ops)
	}
}

func keys(m map[string]map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
