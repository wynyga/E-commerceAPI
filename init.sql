-- Skrip ini akan dijalankan secara otomatis oleh Docker
-- saat container 'db' pertama kali dibuat.

-- Membuat database untuk auth-service
CREATE DATABASE IF NOT EXISTS `auth-service`;

-- Membuat database untuk cart-service
CREATE DATABASE IF NOT EXISTS `cart-service`;

-- Membuat database untuk product-service
CREATE DATABASE IF NOT EXISTS `product-service`;