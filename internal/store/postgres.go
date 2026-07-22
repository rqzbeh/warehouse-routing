package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"warehouse-routing/internal/routing"
)

var ErrStockChanged = errors.New("selected warehouse no longer has enough stock")

type Postgres struct {
	db *pgxpool.Pool
}

func Open(ctx context.Context, dsn string) (*Postgres, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	cfg.MaxConns = 32
	cfg.MinConns = 4
	cfg.HealthCheckPeriod = 30 * time.Second

	db, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return &Postgres{db: db}, nil
}

func (p *Postgres) Close() {
	p.db.Close()
}

func (p *Postgres) Ping(ctx context.Context) error {
	return p.db.Ping(ctx)
}

func (p *Postgres) Candidates(ctx context.Context, sku string, quantity int) ([]routing.Candidate, error) {
	rows, err := p.db.Query(ctx, `
		SELECT w.id, w.lat, w.lon
		FROM warehouses w
		JOIN inventory i ON i.warehouse_id = w.id
		WHERE i.sku = $1 AND i.available_quantity >= $2
	`, sku, quantity)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var candidates []routing.Candidate
	for rows.Next() {
		var c routing.Candidate
		if err := rows.Scan(&c.WarehouseID, &c.Lat, &c.Lon); err != nil {
			return nil, err
		}
		candidates = append(candidates, c)
	}
	return candidates, rows.Err()
}

func (p *Postgres) Reserve(ctx context.Context, warehouseID, sku string, quantity int) error {
	tag, err := p.db.Exec(ctx, `
		UPDATE inventory
		SET available_quantity = available_quantity - $3,
		    reserved_quantity = reserved_quantity + $3,
		    updated_at = now()
		WHERE warehouse_id = $1
		  AND sku = $2
		  AND available_quantity >= $3
	`, warehouseID, sku, quantity)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return ErrStockChanged
	}
	return nil
}
