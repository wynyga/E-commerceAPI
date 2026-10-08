"use client";

import { useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";

import { api } from "@/lib/api";
import { useSession } from "./AuthProvider";
import ThemeToggle from "./ThemeToggle";
import { Button } from "./ui";

function NavLink({ href, children }: { href: string; children: React.ReactNode }) {
  const path = usePathname();
  const active = href === "/" ? path === "/" || path.startsWith("/products") : path.startsWith(href);
  return (
    <Link
      href={href}
      aria-current={active ? "page" : undefined}
      className={`rounded-lg px-2.5 py-1.5 text-sm font-medium ${active ? "bg-surface2 text-ink" : "text-muted hover:text-ink"}`}
    >
      {children}
    </Link>
  );
}

export default function SiteHeader() {
  const { user, signOut } = useSession();
  const router = useRouter();
  const cart = useQuery({ queryKey: ["cart"], queryFn: api.cart.get, enabled: !!user });
  const count = cart.data?.items.reduce((n, i) => n + i.quantity, 0) ?? 0;

  return (
    <header className="sticky top-0 z-30 border-b border-line bg-bg/90 backdrop-blur">
      <div className="mx-auto flex max-w-6xl flex-wrap items-center gap-x-4 gap-y-2 px-4 py-3 sm:px-6">
        <Link href="/" className="font-display text-xl tracking-tight">
          Kedai Seduh
        </Link>
        <nav aria-label="Main" className="flex items-center gap-1">
          <NavLink href="/">Shop</NavLink>
          {user && <NavLink href="/orders">Orders</NavLink>}
          {user?.role === "admin" && <NavLink href="/admin/products">Manage products</NavLink>}
        </nav>

        <div className="ml-auto flex items-center gap-2">
          {user ? (
            <>
              <Link
                href="/cart"
                aria-label={`Cart, ${count} item${count === 1 ? "" : "s"}`}
                className="inline-flex items-center gap-1.5 rounded-lg border border-line bg-surface px-3 py-1.5 text-sm font-medium hover:bg-surface2"
              >
                <svg className="size-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                  <path d="M3 4h2l2.4 11h10.2L20 8H6.2" />
                  <circle cx="9" cy="19.5" r="1.2" />
                  <circle cx="17" cy="19.5" r="1.2" />
                </svg>
                Cart
                {count > 0 && <span className="rounded-full bg-accent px-1.5 text-xs text-accent-ink tabular-nums">{count}</span>}
              </Link>
              <span className="hidden max-w-40 truncate text-sm text-muted md:inline" title={user.email}>
                {user.email}
              </span>
              <Button
                size="sm"
                variant="ghost"
                onClick={async () => {
                  await signOut();
                  router.push("/");
                }}
              >
                Sign out
              </Button>
            </>
          ) : (
            <Link href="/login" className="rounded-lg bg-accent px-3.5 py-2 text-sm font-medium text-accent-ink hover:opacity-90">
              Sign in
            </Link>
          )}
          <ThemeToggle />
        </div>
      </div>
    </header>
  );
}
