INSERT INTO warehouses (id, name, lat, lon) VALUES
    ('tehran-west', 'Tehran West Hub', 35.7219, 51.3347),
    ('tehran-east', 'Tehran East Hub', 35.7390, 51.5330),
    ('karaj', 'Karaj Hub', 35.8327, 50.9916)
ON CONFLICT (id) DO NOTHING;

INSERT INTO inventory (warehouse_id, sku, available_quantity) VALUES
    ('tehran-west', 'SKU-1', 100),
    ('tehran-east', 'SKU-1', 60),
    ('karaj', 'SKU-1', 80),
    ('tehran-west', 'SKU-2', 10)
ON CONFLICT (warehouse_id, sku) DO NOTHING;
