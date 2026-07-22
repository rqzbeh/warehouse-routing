CREATE TABLE IF NOT EXISTS warehouses (
    id text PRIMARY KEY,
    name text NOT NULL,
    lat double precision NOT NULL CHECK (lat BETWEEN -90 AND 90),
    lon double precision NOT NULL CHECK (lon BETWEEN -180 AND 180)
);

CREATE TABLE IF NOT EXISTS inventory (
    warehouse_id text NOT NULL REFERENCES warehouses(id),
    sku text NOT NULL,
    available_quantity integer NOT NULL CHECK (available_quantity >= 0),
    reserved_quantity integer NOT NULL DEFAULT 0 CHECK (reserved_quantity >= 0),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (warehouse_id, sku)
);

CREATE INDEX IF NOT EXISTS warehouses_lat_lon_idx ON warehouses (lat, lon);
CREATE INDEX IF NOT EXISTS inventory_sku_available_idx ON inventory (sku, available_quantity);
