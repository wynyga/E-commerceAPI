import { cookies } from "next/headers";
import { NextResponse, type NextRequest } from "next/server";

import { backend } from "@/lib/server/backend";
import { COOKIE, type Service } from "@/lib/server/session";

// Forwards /api/<anything> to the right Go service, adding the session token from the httpOnly
// cookie. The browser never talks to the services directly (they send no CORS headers) and
// never sees the token. /api/auth/login and /api/auth/logout have their own routes because
// they manage the cookie, and take precedence over this catch-all.

type Ctx = { params: Promise<{ path: string[] }> };

/** Which service owns a path, and what the path is called there. */
function route(path: string[]): { service: Service; upstream: string } | null {
  const [head, ...rest] = path;
  const tail = rest.map(encodeURIComponent).join("/");
  const join = (base: string) => `/api/v1/${base}${tail ? `/${tail}` : ""}`;
  switch (head) {
    case "products":
      return { service: "product", upstream: join("products") };
    case "cart":
      return { service: "cart", upstream: join("cart") };
    case "orders":
      return { service: "order", upstream: join("orders") };
    case "profile":
      return rest.length === 0 ? { service: "auth", upstream: "/api/v1/profile" } : null;
    case "auth":
      return rest.length === 1 && rest[0] === "register" ? { service: "auth", upstream: "/api/v1/register" } : null;
    default:
      return null;
  }
}

async function proxy(req: NextRequest, ctx: Ctx) {
  const { path } = await ctx.params;
  const target = route(path);
  if (!target) return NextResponse.json({ error: "not found" }, { status: 404 });

  const token = (await cookies()).get(COOKIE)?.value;
  const hasBody = req.method !== "GET" && req.method !== "HEAD";
  const key = req.headers.get("idempotency-key");

  const res = await backend(target.service, req.method, target.upstream, {
    search: req.nextUrl.search,
    body: hasBody ? await req.text() : undefined,
    token,
    headers: key ? { "Idempotency-Key": key } : undefined,
  });
  if (res.status === 204 || res.json === null) return new NextResponse(null, { status: res.status });
  return NextResponse.json(res.json, { status: res.status });
}

export { proxy as GET, proxy as POST, proxy as PUT, proxy as PATCH, proxy as DELETE };
