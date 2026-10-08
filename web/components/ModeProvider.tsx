"use client";

import { useQueryClient } from "@tanstack/react-query";
import { useState } from "react";

import { configureApi } from "@/lib/api";
import { resetDemo } from "@/lib/demo/handlers";

/**
 * Tells the API client whether to talk to the services or to the in-browser demo. It does so during
 * render, so the setting is in place before any child runs an effect that fetches data.
 */
export default function ModeProvider({ demo, children }: { demo: boolean; children: React.ReactNode }) {
  configureApi({ demo });
  const qc = useQueryClient();
  const [confirming, setConfirming] = useState(false);

  return (
    <>
      {children}
      {demo && (
        <div className="fixed bottom-3 left-3 z-[99999] flex max-w-[calc(100vw-1.5rem)] flex-wrap items-center gap-x-2 gap-y-1 rounded-2xl border border-line bg-surface px-3 py-1.5 text-xs text-ink shadow-lg">
          <span className="size-2 rounded-full bg-accent" aria-hidden="true" />
          <span className="font-medium">Demo mode</span>
          <span className="hidden text-muted sm:inline">· the shop runs in your browser, nothing is sent anywhere</span>
          {confirming ? (
            <>
              <button
                type="button"
                className="rounded-full bg-accent px-2 py-0.5 font-medium text-accent-ink"
                onClick={() => {
                  resetDemo();
                  qc.clear();
                  window.location.assign("/");
                }}
              >
                Yes, reset
              </button>
              <button type="button" className="underline" onClick={() => setConfirming(false)}>
                Cancel
              </button>
            </>
          ) : (
            <button type="button" className="underline" onClick={() => setConfirming(true)}>
              Reset shop
            </button>
          )}
        </div>
      )}
    </>
  );
}
