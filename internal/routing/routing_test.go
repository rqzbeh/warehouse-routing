package routing

import (
	"errors"
	"testing"
)

func TestSelectChoosesLowestAdjustedCost(t *testing.T) {
	req := Request{
		CustomerLocation: Location{Lat: 35.7219, Lon: 51.3347},
		SKU:              "SKU-1",
		Quantity:         2,
		TransportationCosts: []TransportationCost{
			{WarehouseID: "far-cheap", Cost: 100_000, ETAMinutes: 90},
			{WarehouseID: "near-expensive", Cost: 250_000, ETAMinutes: 45},
		},
		LogisticsConstraints: []LogisticsConstraint{
			{WarehouseID: "far-cheap", TrafficCoefficient: 1.1, FleetPriorityFactor: 1},
			{WarehouseID: "near-expensive", TrafficCoefficient: 1, FleetPriorityFactor: 1},
		},
	}
	candidates := []Candidate{
		{WarehouseID: "far-cheap", Lat: 35.6892, Lon: 51.3890},
		{WarehouseID: "near-expensive", Lat: 35.7219, Lon: 51.3347},
	}

	got, err := Select(req, candidates)
	if err != nil {
		t.Fatal(err)
	}
	if got.WarehouseID != "far-cheap" {
		t.Fatalf("warehouse = %q, want far-cheap", got.WarehouseID)
	}
	if got.EstimatedDeliveryMinutes != 100 {
		t.Fatalf("eta = %d, want 100", got.EstimatedDeliveryMinutes)
	}
	if got.TotalCost <= 0 || got.RouteOptimizationScore <= 0 {
		t.Fatalf("expected positive cost and score, got %+v", got)
	}
}

func TestSelectSkipsCandidatesWithoutTransportCost(t *testing.T) {
	req := Request{
		CustomerLocation:    Location{Lat: 35, Lon: 51},
		SKU:                 "SKU-1",
		Quantity:            1,
		TransportationCosts: []TransportationCost{{WarehouseID: "w2", Cost: 1, ETAMinutes: 1}},
	}

	_, err := Select(req, []Candidate{{WarehouseID: "w1", Lat: 35, Lon: 51}})
	if !errors.Is(err, ErrNoCandidate) {
		t.Fatalf("err = %v, want ErrNoCandidate", err)
	}
}

func TestSelectMatchesReadmeExample(t *testing.T) {
	req := Request{
		CustomerLocation: Location{Lat: 35.7219, Lon: 51.3347},
		SKU:              "SKU-1",
		Quantity:         2,
		TransportationCosts: []TransportationCost{
			{WarehouseID: "tehran-west", Cost: 100000, ETAMinutes: 45},
			{WarehouseID: "tehran-east", Cost: 85000, ETAMinutes: 70},
			{WarehouseID: "karaj", Cost: 70000, ETAMinutes: 95},
		},
		LogisticsConstraints: []LogisticsConstraint{
			{WarehouseID: "tehran-west", TrafficCoefficient: 1.2, FleetPriorityFactor: 1.1},
			{WarehouseID: "tehran-east", TrafficCoefficient: 1.1, FleetPriorityFactor: 1},
			{WarehouseID: "karaj", TrafficCoefficient: 1, FleetPriorityFactor: 0.9},
		},
	}
	candidates := []Candidate{
		{WarehouseID: "tehran-west", Lat: 35.7219, Lon: 51.3347},
		{WarehouseID: "tehran-east", Lat: 35.7390, Lon: 51.5330},
		{WarehouseID: "karaj", Lat: 35.8327, Lon: 50.9916},
	}

	got, err := Select(req, candidates)
	if err != nil {
		t.Fatal(err)
	}
	if got.WarehouseID != "tehran-west" {
		t.Fatalf("warehouse = %q, want tehran-west", got.WarehouseID)
	}
	if got.EstimatedDeliveryMinutes != 50 {
		t.Fatalf("eta = %d, want 50", got.EstimatedDeliveryMinutes)
	}
}

func TestSelectPenalizesLateDelivery(t *testing.T) {
	req := Request{
		CustomerLocation:     Location{Lat: 35, Lon: 51},
		SKU:                  "SKU-1",
		Quantity:             1,
		ExpectedDeliveryTime: 30,
		TransportationCosts: []TransportationCost{
			{WarehouseID: "cheap-late", Cost: 100, ETAMinutes: 60},
			{WarehouseID: "expensive-fast", Cost: 200, ETAMinutes: 20},
		},
	}

	got, err := Select(req, []Candidate{
		{WarehouseID: "cheap-late", Lat: 35, Lon: 51},
		{WarehouseID: "expensive-fast", Lat: 35, Lon: 51},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.WarehouseID != "expensive-fast" {
		t.Fatalf("warehouse = %q, want expensive-fast", got.WarehouseID)
	}
}

func TestValidateRejectsBadLocation(t *testing.T) {
	err := Validate(Request{CustomerLocation: Location{Lat: 100, Lon: 51}, SKU: "SKU-1", Quantity: 1})
	if !errors.Is(err, ErrBadRequest) {
		t.Fatalf("err = %v, want ErrBadRequest", err)
	}
}
