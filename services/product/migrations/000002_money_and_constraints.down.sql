ALTER TABLE products
    DROP CONSTRAINT products_stock_nonneg,
    DROP INDEX products_name_idx,
    MODIFY COLUMN price DECIMAL(10, 2) NOT NULL;
