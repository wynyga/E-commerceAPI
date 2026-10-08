"use client";

import { usePathname, useRouter } from "next/navigation";
import { useEffect } from "react";

import type { Role } from "@/lib/types";
import { useSession } from "./AuthProvider";
import { Card, EmptyState, SkeletonRows } from "./ui";

/** Shows its children only to a signed-in user (with the given role); otherwise sends them to sign in. */
export default function RequireAuth({ role, children }: { role?: Role; children: React.ReactNode }) {
  const { user, loading } = useSession();
  const router = useRouter();
  const path = usePathname();

  useEffect(() => {
    if (!loading && !user) router.replace(`/login?next=${encodeURIComponent(path)}`);
  }, [loading, user, router, path]);

  if (loading || !user) {
    return (
      <Card>
        <SkeletonRows rows={4} />
      </Card>
    );
  }
  if (role && user.role !== role) {
    return (
      <Card>
        <EmptyState title="Admins only" hint={`This page needs the ${role} role, and you are signed in as a ${user.role}.`} />
      </Card>
    );
  }
  return <>{children}</>;
}
