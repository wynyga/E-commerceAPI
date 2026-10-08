import { cookies } from "next/headers";
import { NextResponse, type NextRequest } from "next/server";

import { backend } from "@/lib/server/backend";
import { COOKIE } from "@/lib/server/session";

// Logs in against the auth service, then keeps the token in an httpOnly cookie so page scripts
// (and anything injected into them) can never read it. The browser only gets the user back.
export async function POST(req: NextRequest) {
  const res = await backend("auth", "POST", "/api/v1/login", { body: await req.text() });
  if (res.status !== 200) return NextResponse.json(res.json, { status: res.status });

  const { token, expires_at, user } = res.json as { token: string; expires_at: string; user: unknown };
  const maxAge = Math.max(60, Math.floor((new Date(expires_at).getTime() - Date.now()) / 1000));
  (await cookies()).set(COOKIE, token, {
    httpOnly: true,
    sameSite: "lax",
    secure: process.env.NODE_ENV === "production",
    path: "/",
    maxAge,
  });
  return NextResponse.json({ user });
}
