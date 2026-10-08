import "server-only";

export const COOKIE = "shop_session";

const clean = (v: string | undefined) => v?.trim().replace(/\/+$/, "") || undefined;

/** Base URLs of the four Go services. If any is unset the app runs in demo mode. */
export const SERVICES = {
  auth: clean(process.env.SHOP_AUTH_URL),
  product: clean(process.env.SHOP_PRODUCT_URL),
  cart: clean(process.env.SHOP_CART_URL),
  order: clean(process.env.SHOP_ORDER_URL),
};

export type Service = keyof typeof SERVICES;

export const isDemo = !(SERVICES.auth && SERVICES.product && SERVICES.cart && SERVICES.order);
