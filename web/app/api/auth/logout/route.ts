import { cookies } from "next/headers";
import { NextResponse } from "next/server";

import { COOKIE } from "@/lib/server/session";

export async function POST() {
  (await cookies()).delete(COOKIE);
  return NextResponse.json({ ok: true });
}
