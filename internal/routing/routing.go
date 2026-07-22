package routing

import (
	"errors"
	"math"
	"time"
)

var (
	ErrNoCandidate = errors.New("no warehouse can fulfill the request")
	ErrBadRequest  = errors.New("invalid route request")
)

type Location struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type Request struct {
	CustomerLocation     Location              `json:"customer_location"`
	SKU                  string                `json:"sku"`
	Quantity             int                   `json:"quantity"`
	RequestedAt          time.Time             `json:"requested_at"`
	ExpectedDeliveryTime int                   `json:"expected_delivery_time"`
	TransportationCosts  []TransportationCost  `json:"transportation_costs"`
	LogisticsConstraints []LogisticsConstraint `json:"logistics_constraints"`
}

type TransportationCost struct {
	WarehouseID string  `json:"warehouse_id"`
	Cost        float64 `json:"cost"`
	ETAMinutes  int     `json:"eta_minutes"`
}

type LogisticsConstraint struct {
	WarehouseID         string  `json:"warehouse_id"`
	TrafficCoefficient  float64 `json:"traffic_coefficient"`
	FleetPriorityFactor float64 `json:"fleet_priority_factor"`
}

type Candidate struct {
	WarehouseID string
	Lat         float64
	Lon         float64
}

type Decision struct {
	WarehouseID              string  `json:"WarehouseID"`
	EstimatedDeliveryMinutes int     `json:"EstimatedDeliveryTime"`
	TotalCost                float64 `json:"TotalCost"`
	RouteOptimizationScore   float64 `json:"RouteOptimizationScore"`
}

func Validate(req Request) error {
	if req.SKU == "" || req.Quantity <= 0 {
		return ErrBadRequest
	}
	if req.CustomerLocation.Lat < -90 || req.CustomerLocation.Lat > 90 ||
		req.CustomerLocation.Lon < -180 || req.CustomerLocation.Lon > 180 {
		return ErrBadRequest
	}
	for _, c := range req.TransportationCosts {
		if c.WarehouseID == "" || c.Cost < 0 || c.ETAMinutes <= 0 {
			return ErrBadRequest
		}
	}
	for _, c := range req.LogisticsConstraints {
		if c.WarehouseID == "" || c.TrafficCoefficient < 0 || c.FleetPriorityFactor < 0 {
			return ErrBadRequest
		}
	}
	return nil
}

func Select(req Request, candidates []Candidate) (Decision, error) {
	if err := Validate(req); err != nil {
		return Decision{}, err
	}
	if len(candidates) == 0 {
		return Decision{}, ErrNoCandidate
	}

	costs := map[string]TransportationCost{}
	for _, c := range req.TransportationCosts {
		costs[c.WarehouseID] = c
	}

	constraints := map[string]LogisticsConstraint{}
	for _, c := range req.LogisticsConstraints {
		constraints[c.WarehouseID] = c
	}

	var best Decision
	bestScore := math.Inf(1)
	for _, candidate := range candidates {
		transport, ok := costs[candidate.WarehouseID]
		if !ok {
			continue
		}

		constraint := constraints[candidate.WarehouseID]
		traffic := defaultIfZero(constraint.TrafficCoefficient, 1)
		fleet := defaultIfZero(constraint.FleetPriorityFactor, 1)
		distanceKM := haversineKM(req.CustomerLocation, Location{Lat: candidate.Lat, Lon: candidate.Lon})
		eta := int(math.Ceil(float64(transport.ETAMinutes) * traffic / fleet))
		delayPenalty := math.Max(0, float64(eta-req.ExpectedDeliveryTime)) * 500
		if req.ExpectedDeliveryTime == 0 {
			delayPenalty = 0
		}
		total := transport.Cost*traffic/fleet + distanceKM*1000 + float64(eta)*100 + delayPenalty
		score := 100 / (1 + total/100000)

		if total < bestScore {
			bestScore = total
			best = Decision{
				WarehouseID:              candidate.WarehouseID,
				EstimatedDeliveryMinutes: eta,
				TotalCost:                round2(total),
				RouteOptimizationScore:   round2(score),
			}
		}
	}
	if best.WarehouseID == "" {
		return Decision{}, ErrNoCandidate
	}
	return best, nil
}

func defaultIfZero(v, fallback float64) float64 {
	if v == 0 {
		return fallback
	}
	return v
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

func haversineKM(a, b Location) float64 {
	const earthKM = 6371
	lat1 := a.Lat * math.Pi / 180
	lat2 := b.Lat * math.Pi / 180
	dLat := (b.Lat - a.Lat) * math.Pi / 180
	dLon := (b.Lon - a.Lon) * math.Pi / 180

	x := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1)*math.Cos(lat2)*math.Sin(dLon/2)*math.Sin(dLon/2)
	return earthKM * 2 * math.Atan2(math.Sqrt(x), math.Sqrt(1-x))
}
