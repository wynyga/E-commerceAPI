import type { Metadata, Viewport } from "next";
import { Fraunces, Inter } from "next/font/google";

import AuthProvider from "@/components/AuthProvider";
import ModeProvider from "@/components/ModeProvider";
import Providers from "@/components/Providers";
import SiteFooter from "@/components/SiteFooter";
import SiteHeader from "@/components/SiteHeader";
import { isDemo } from "@/lib/server/session";
import "./globals.css";

const sans = Inter({ subsets: ["latin"], variable: "--font-inter", display: "swap" });
const display = Fraunces({ subsets: ["latin"], variable: "--font-fraunces", display: "swap" });

export const metadata: Metadata = {
  title: { default: "Kedai Seduh", template: "%s · Kedai Seduh" },
  description: "A small coffee gear shop built on four Go microservices: accounts, catalogue, cart and orders.",
};

export const viewport: Viewport = {
  colorScheme: "light dark",
  themeColor: [
    { media: "(prefers-color-scheme: light)", color: "#faf7f2" },
    { media: "(prefers-color-scheme: dark)", color: "#14110f" },
  ],
};

// Runs before first paint so the saved (or system) theme never flashes.
const themeScript = `(function(){try{var t=localStorage.getItem('theme');if(t!=='light'&&t!=='dark'){t=matchMedia('(prefers-color-scheme: dark)').matches?'dark':'light'}document.documentElement.dataset.theme=t}catch(e){}})();`;

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" className={`${sans.variable} ${display.variable}`} suppressHydrationWarning>
      <head>
        <script dangerouslySetInnerHTML={{ __html: themeScript }} />
      </head>
      <body className="min-h-dvh font-sans antialiased">
        <Providers>
          <ModeProvider demo={isDemo}>
            <AuthProvider>
              <SiteHeader />
              <main className="mx-auto max-w-6xl px-4 pt-8 pb-4 sm:px-6">{children}</main>
              <SiteFooter />
            </AuthProvider>
          </ModeProvider>
        </Providers>
      </body>
    </html>
  );
}
