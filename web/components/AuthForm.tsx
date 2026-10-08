"use client";

import { useRouter, useSearchParams } from "next/navigation";
import { useEffect, useState } from "react";

import { api } from "@/lib/api";
import { DEMO_ADMIN, DEMO_CUSTOMER } from "@/lib/demo/accounts";
import { useAction } from "@/lib/hooks";
import { useSession } from "./AuthProvider";
import { Button, Card, TextField } from "./ui";

/** Only follow same-site paths, never an address someone put in the link. */
function safeNext(v: string | null): string {
  return v && v.startsWith("/") && !v.startsWith("//") && !v.startsWith("/\\") ? v : "/";
}

export default function AuthForm({ demo }: { demo: boolean }) {
  const router = useRouter();
  const next = safeNext(useSearchParams().get("next"));
  const { user, signedIn } = useSession();

  const [mode, setMode] = useState<"login" | "register">("login");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [fields, setFields] = useState<Record<string, string>>({});
  const [formError, setFormError] = useState("");

  useEffect(() => {
    if (user) router.replace(next);
  }, [user, next, router]);

  const login = useAction(api.login, { toastError: false });
  const register = useAction(api.register, { toastError: false });
  const pending = login.pending || register.pending;

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setFields({});
    setFormError("");

    if (mode === "register") {
      const created = await register.run(email, password);
      if (!created.ok) return fail(created.error);
    }
    const res = await login.run(email, password);
    if (!res.ok) return fail(res.error);
    signedIn(res.data.user);
    router.replace(next);
  }

  function fail(error: { message: string; fields?: Record<string, string> }) {
    if (error.fields) setFields(error.fields);
    else setFormError(error.message);
  }

  function fill(c: { email: string; password: string }) {
    setMode("login");
    setEmail(c.email);
    setPassword(c.password);
    setFields({});
    setFormError("");
  }

  return (
    <div className="mx-auto max-w-md">
      <h1 className="font-display text-3xl leading-tight sm:text-4xl">{mode === "login" ? "Sign in" : "Create an account"}</h1>
      <p className="mt-1 mb-6 text-sm text-muted">
        {mode === "login" ? "Sign in to fill a cart and place orders." : "Accounts are customers. Admins are created by the server, never through this form."}
      </p>

      <Card className="p-6">
        <form onSubmit={submit} noValidate className="space-y-4">
          <TextField label="Email" type="email" autoComplete="email" value={email} onChange={(e) => setEmail(e.target.value)} error={fields.email} />
          <TextField
            label="Password"
            type="password"
            autoComplete={mode === "login" ? "current-password" : "new-password"}
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            error={fields.password}
            hint={mode === "register" ? "At least 8 characters." : undefined}
          />
          {formError && (
            <p role="alert" className="text-sm text-danger">
              {formError}
            </p>
          )}
          <Button type="submit" variant="primary" className="w-full" loading={pending}>
            {mode === "login" ? "Sign in" : "Create account"}
          </Button>
        </form>

        <p className="mt-4 text-center text-sm text-muted">
          {mode === "login" ? "New here?" : "Already have an account?"}{" "}
          <button
            type="button"
            className="underline hover:text-ink"
            onClick={() => {
              setMode(mode === "login" ? "register" : "login");
              setFields({});
              setFormError("");
            }}
          >
            {mode === "login" ? "Create an account" : "Sign in"}
          </button>
        </p>
      </Card>

      {demo && (
        <Card className="mt-4 border-dashed p-5">
          <p className="text-sm font-medium">Demo accounts</p>
          <p className="mt-1 text-xs text-muted">One click fills the form. The admin can add, edit and remove products.</p>
          <div className="mt-3 flex flex-wrap gap-2">
            <Button size="sm" onClick={() => fill(DEMO_CUSTOMER)}>
              Use the customer
            </Button>
            <Button size="sm" onClick={() => fill(DEMO_ADMIN)}>
              Use the admin
            </Button>
          </div>
        </Card>
      )}
    </div>
  );
}
