#!/bin/sh
# Runs once, the first time the MySQL volume is created.
# Each service owns its own database, and the services connect as a dedicated user that
# can touch those four databases only (never as root).
set -e

mysql -uroot -p"$MYSQL_ROOT_PASSWORD" <<SQL
CREATE DATABASE IF NOT EXISTS shop_auth;
CREATE DATABASE IF NOT EXISTS shop_product;
CREATE DATABASE IF NOT EXISTS shop_cart;
CREATE DATABASE IF NOT EXISTS shop_order;

CREATE USER IF NOT EXISTS 'shop'@'%' IDENTIFIED BY '${APP_DB_PASSWORD}';
GRANT ALL PRIVILEGES ON shop_auth.*    TO 'shop'@'%';
GRANT ALL PRIVILEGES ON shop_product.* TO 'shop'@'%';
GRANT ALL PRIVILEGES ON shop_cart.*    TO 'shop'@'%';
GRANT ALL PRIVILEGES ON shop_order.*   TO 'shop'@'%';
FLUSH PRIVILEGES;
SQL
