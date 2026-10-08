-- The first version looked a cart or an item up and then inserted it, so two simultaneous
-- requests could create duplicates. Clean up any that exist, then let the database forbid them.
DELETE c1 FROM carts c1 JOIN carts c2 ON c1.user_id = c2.user_id AND c1.id > c2.id;
DELETE i1 FROM cart_items i1 JOIN cart_items i2
    ON i1.cart_id = i2.cart_id AND i1.product_id = i2.product_id AND i1.id > i2.id;

ALTER TABLE carts
    ADD UNIQUE KEY carts_user_uq (user_id);

ALTER TABLE cart_items
    ADD UNIQUE KEY cart_items_cart_product_uq (cart_id, product_id),
    ADD CONSTRAINT cart_items_qty_positive CHECK (quantity > 0);
