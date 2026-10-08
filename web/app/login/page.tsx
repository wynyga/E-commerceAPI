import { Suspense } from "react";

import AuthForm from "@/components/AuthForm";
import { isDemo } from "@/lib/server/session";

export const metadata = { title: "Sign in" };

export default function LoginPage() {
  return (
    <Suspense>
      <AuthForm demo={isDemo} />
    </Suspense>
  );
}
