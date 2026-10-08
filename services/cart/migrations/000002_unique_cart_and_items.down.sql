ALTER TABLE cart_items
    DROP CONSTRAINT cart_items_qty_positive,
    DROP INDEX cart_items_cart_product_uq;

ALTER TABLE carts
    DROP INDEX carts_user_uq;
