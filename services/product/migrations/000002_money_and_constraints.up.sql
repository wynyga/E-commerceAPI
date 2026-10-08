-- Prices are whole rupiah (IDR has no minor unit), stored as integers: no float or decimal
-- rounding when totals are added up.
ALTER TABLE products
    MODIFY COLUMN price BIGINT UNSIGNED NOT NULL,
    ADD INDEX products_name_idx (name),
    ADD CONSTRAINT products_stock_nonneg CHECK (stock >= 0);
