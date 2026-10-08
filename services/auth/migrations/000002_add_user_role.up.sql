-- Customers are the default; admins are created at startup from ADMIN_EMAIL / ADMIN_PASSWORD.
ALTER TABLE users
    ADD COLUMN `role` VARCHAR(20) NOT NULL DEFAULT 'customer' AFTER `password`,
    ADD CONSTRAINT users_role_chk CHECK (`role` IN ('customer', 'admin'));
