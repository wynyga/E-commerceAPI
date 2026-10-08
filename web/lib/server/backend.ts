import "server-only";

import { SERVICES, type Service } from "./session";

export type BackendResult = { status: number; json: unknown };

/** Calls one of the Go services and returns its status and JSON body (always JSON, even when the service is down). */
export async function backend(
  service: Service,
  method: string,
  path: string,
  opts: { search?: string; body?: string; token?: string; headers?: Record<string, string> } = {},
): Promise<BackendResult> {
  const base = SERVICES[service];
  if (!base) return { status: 503, json: { error: `The ${service} service is not configured.` } };

  const headers: Record<string, string> = { ...opts.headers };
  if (opts.token) headers.Authorization = `Bearer ${opts.token}`;
  if (opts.body) headers["Content-Type"] = "application/json";

  try {
    const res = await fetch(`${base}${path}${opts.search ?? ""}`, {
      method,
      headers,
      body: opts.body,
      cache: "no-store",
      signal: AbortSignal.timeout(15_000),
    });
    const text = await res.text();
    if (!text) return { status: res.status, json: res.ok ? null : { error: res.statusText || "The service returned an empty response." } };
    try {
      return { status: res.status, json: JSON.parse(text) };
    } catch {
      return { status: 502, json: { error: "The service returned an unexpected response." } };
    }
  } catch {
    return { status: 502, json: { error: `Cannot reach the ${service} service.` } };
  }
}
