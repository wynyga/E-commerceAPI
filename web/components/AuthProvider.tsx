"use client";

import { useQuery, useQueryClient } from "@tanstack/react-query";
import { createContext, useCallback, useContext, useEffect, useMemo } from "react";

import { api, ApiError, UNAUTHORIZED_EVENT } from "@/lib/api";
import type { User } from "@/lib/types";

type Session = {
  user: User | null;
  /** True until the first answer about who is signed in. */
  loading: boolean;
  signedIn: (user: User) => void;
  signOut: () => Promise<void>;
};

const Ctx = createContext<Session>({ user: null, loading: true, signedIn: () => {}, signOut: async () => {} });

export const useSession = () => useContext(Ctx);

/** The catalogue is the same for everyone; everything else cached belongs to whoever was signed in. ("me" is set explicitly.) */
const isPersonal = (key: readonly unknown[]) => !["products", "product", "me"].includes(String(key[0]));

export default function AuthProvider({ children }: { children: React.ReactNode }) {
  const qc = useQueryClient();

  const me = useQuery({
    queryKey: ["me"],
    queryFn: async (): Promise<User | null> => {
      try {
        return await api.me();
      } catch (e) {
        if (e instanceof ApiError && e.status === 401) return null;
        throw e;
      }
    },
    staleTime: 60_000,
  });

  // A signed-in call came back 401 (expired token): forget the user so the UI asks to sign in again.
  useEffect(() => {
    const onUnauthorized = () => qc.setQueryData(["me"], null);
    window.addEventListener(UNAUTHORIZED_EVENT, onUnauthorized);
    return () => window.removeEventListener(UNAUTHORIZED_EVENT, onUnauthorized);
  }, [qc]);

  const signedIn = useCallback(
    (user: User) => {
      qc.removeQueries({ predicate: (q) => isPersonal(q.queryKey) });
      qc.setQueryData(["me"], user);
    },
    [qc],
  );

  const signOut = useCallback(async () => {
    await api.logout();
    qc.removeQueries({ predicate: (q) => isPersonal(q.queryKey) });
    qc.setQueryData(["me"], null);
  }, [qc]);

  const value = useMemo<Session>(
    () => ({ user: me.data ?? null, loading: me.isPending, signedIn, signOut }),
    [me.data, me.isPending, signedIn, signOut],
  );

  return <Ctx.Provider value={value}>{children}</Ctx.Provider>;
}
