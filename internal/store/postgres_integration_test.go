package store

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"
)

func TestPostgresCandidatesAndReserve(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := Open(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(db.Close)

	_, err = db.db.Exec(ctx, `
		INSERT INTO warehouses (id, name, lat, lon)
		VALUES ('test-store-w1', 'Test Store W1', 35, 51)
		ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name;

		INSERT INTO inventory (warehouse_id, sku, available_quantity, reserved_quantity)
		VALUES ('test-store-w1', 'TEST-SKU', 2, 0)
		ON CONFLICT (warehouse_id, sku)
		DO UPDATE SET available_quantity = 2, reserved_quantity = 0;
	`)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		_, _ = db.db.Exec(cleanupCtx, `
			DELETE FROM inventory WHERE warehouse_id = 'test-store-w1';
			DELETE FROM warehouses WHERE id = 'test-store-w1';
		`)
	})

	candidates, err := db.Candidates(ctx, "TEST-SKU", 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 1 || candidates[0].WarehouseID != "test-store-w1" {
		t.Fatalf("candidates = %+v, want test-store-w1", candidates)
	}

	if err := db.Reserve(ctx, "test-store-w1", "TEST-SKU", 2); err != nil {
		t.Fatal(err)
	}
	if err := db.Reserve(ctx, "test-store-w1", "TEST-SKU", 1); !errors.Is(err, ErrStockChanged) {
		t.Fatalf("err = %v, want ErrStockChanged", err)
	}
}

func TestPostgresConcurrentReserveAllowsOnlyAvailableStock(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := Open(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(db.Close)

	_, err = db.db.Exec(ctx, `
		INSERT INTO warehouses (id, name, lat, lon)
		VALUES ('test-store-w2', 'Test Store W2', 35, 51)
		ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name;

		INSERT INTO inventory (warehouse_id, sku, available_quantity, reserved_quantity)
		VALUES ('test-store-w2', 'TEST-CONCURRENT-SKU', 1, 0)
		ON CONFLICT (warehouse_id, sku)
		DO UPDATE SET available_quantity = 1, reserved_quantity = 0;
	`)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		_, _ = db.db.Exec(cleanupCtx, `
			DELETE FROM inventory WHERE warehouse_id = 'test-store-w2';
			DELETE FROM warehouses WHERE id = 'test-store-w2';
		`)
	})

	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- db.Reserve(context.Background(), "test-store-w2", "TEST-CONCURRENT-SKU", 1)
		}()
	}
	wg.Wait()
	close(errs)

	var ok, changed int
	for err := range errs {
		switch {
		case err == nil:
			ok++
		case errors.Is(err, ErrStockChanged):
			changed++
		default:
			t.Fatal(err)
		}
	}
	if ok != 1 || changed != 1 {
		t.Fatalf("successes = %d, stock_changed = %d, want 1 and 1", ok, changed)
	}
}
