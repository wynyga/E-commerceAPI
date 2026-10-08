-- unit_price and name are snapshots taken when the stock is reserved, so an order keeps
-- showing what the customer was charged even if the product is later renamed or repriced.
CREATE TABLE orders (
    id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    user_id         BIGINT UNSIGNED NOT NULL,
    status          VARCHAR(20)     NOT NULL DEFAULT 'pending',
    total           BIGINT UNSIGNED NOT NULL,
    idempotency_key VARCHAR(100)    NULL,
    created_at      TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX orders_user_idx (user_id, id),
    -- A repeated checkout with the same key must not create a second order. NULL keys are not compared.
    UNIQUE KEY orders_idempotency_uq (user_id, idempotency_key),
    CONSTRAINT orders_status_chk CHECK (status IN ('pending', 'cancelled'))
) ENGINE=InnoDB;

CREATE TABLE order_items (
    id         BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    order_id   BIGINT UNSIGNED NOT NULL,
    product_id BIGINT UNSIGNED NOT NULL,
    name       VARCHAR(255)    NOT NULL,
    unit_price BIGINT UNSIGNED NOT NULL,
    quantity   INT             NOT NULL,
    INDEX order_items_order_idx (order_id),
    CONSTRAINT order_items_order_fk FOREIGN KEY (order_id) REFERENCES orders (id) ON DELETE CASCADE,
    CONSTRAINT order_items_qty_positive CHECK (quantity > 0)
) ENGINE=InnoDB;
