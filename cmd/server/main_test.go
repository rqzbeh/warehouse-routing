package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"warehouse-routing/internal/routing"
)

type fakeStore struct {
	candidates []routing.Candidate
	reserved   string
	err        error
}

func (f *fakeStore) Candidates(context.Context, string, int) ([]routing.Candidate, error) {
	return f.candidates, f.err
}

func (f *fakeStore) Reserve(_ context.Context, warehouseID, _ string, _ int) error {
	f.reserved = warehouseID
	return f.err
}

func (f *fakeStore) Ping(context.Context) error {
	return f.err
}

func TestReadyReportsHealthyStore(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)

	app{store: &fakeStore{}}.ready(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
}

func TestReadyReportsStoreFailure(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)

	app{store: &fakeStore{err: errors.New("db down")}}.ready(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
}

func TestRouteReturnsDecisionAndReservesStock(t *testing.T) {
	store := &fakeStore{candidates: []routing.Candidate{{WarehouseID: "w1", Lat: 35, Lon: 51}}}
	body := `{
		"customer_location":{"lat":35,"lon":51},
		"sku":"SKU-1",
		"quantity":1,
		"transportation_costs":[{"warehouse_id":"w1","cost":100,"eta_minutes":10}]
	}`

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/route", strings.NewReader(body))
	app{store: store}.route(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if store.reserved != "w1" {
		t.Fatalf("reserved = %q, want w1", store.reserved)
	}

	var got routing.Decision
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.WarehouseID != "w1" {
		t.Fatalf("warehouse = %q, want w1", got.WarehouseID)
	}
}

func TestRouteUsesPDFOutputFieldNames(t *testing.T) {
	store := &fakeStore{candidates: []routing.Candidate{{WarehouseID: "w1", Lat: 35, Lon: 51}}}
	body := `{
		"customer_location":{"lat":35,"lon":51},
		"sku":"SKU-1",
		"quantity":1,
		"transportation_costs":[{"warehouse_id":"w1","cost":100,"eta_minutes":10}]
	}`

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/route", strings.NewReader(body))
	app{store: store}.route(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	for _, field := range []string{"WarehouseID", "EstimatedDeliveryTime", "TotalCost", "RouteOptimizationScore"} {
		if !strings.Contains(rec.Body.String(), `"`+field+`"`) {
			t.Fatalf("response does not contain %s: %s", field, rec.Body.String())
		}
	}
}

func TestRouteRejectsUnknownJSONFields(t *testing.T) {
	body := `{"sku":"SKU-1","quantity":1,"customer_location":{"lat":35,"lon":51},"extra":true}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/route", strings.NewReader(body))

	app{store: &fakeStore{}}.route(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestRouteRejectsTrailingJSON(t *testing.T) {
	body := `{"sku":"SKU-1","quantity":1,"customer_location":{"lat":35,"lon":51},"transportation_costs":[{"warehouse_id":"w1","cost":100,"eta_minutes":10}]}{}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/route", strings.NewReader(body))

	app{store: &fakeStore{}}.route(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestRouteReportsNoWarehouse(t *testing.T) {
	body := `{
		"customer_location":{"lat":35,"lon":51},
		"sku":"SKU-1",
		"quantity":1,
		"transportation_costs":[{"warehouse_id":"w1","cost":100,"eta_minutes":10}]
	}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/route", strings.NewReader(body))

	app{store: &fakeStore{}}.route(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestRouteReportsCandidateLookupFailure(t *testing.T) {
	body := `{
		"customer_location":{"lat":35,"lon":51},
		"sku":"SKU-1",
		"quantity":1,
		"transportation_costs":[{"warehouse_id":"w1","cost":100,"eta_minutes":10}]
	}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/route", strings.NewReader(body))

	app{store: &fakeStore{err: errors.New("db down")}}.route(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
}

func TestRouteRejectsMissingTransportationCosts(t *testing.T) {
	body := `{"customer_location":{"lat":35,"lon":51},"sku":"SKU-1","quantity":1}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/route", strings.NewReader(body))

	app{store: &fakeStore{}}.route(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}
