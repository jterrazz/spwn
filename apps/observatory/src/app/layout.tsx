import type { Metadata } from "next";
import { Geist, Geist_Mono, Space_Grotesk } from "next/font/google";
import "./globals.css";
import { Aurora } from "@/components/aurora";
import { Stars } from "@/components/stars";
import { ThemeProvider } from "@/components/theme-provider";
import { TooltipProvider } from "@/components/ui/tooltip";
import { ToastProvider } from "@/components/toast-provider";
import { AppShell } from "@/components/app-shell";
import { CommandPalette } from "@/components/command-palette";

const geistSans = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

const spaceGrotesk = Space_Grotesk({
  variable: "--font-space-grotesk",
  subsets: ["latin"],
  weight: ["400", "500", "600", "700"],
});

export const metadata: Metadata = {
  title: {
    template: '%s · Observatory',
    default: 'Observatory · spwn',
  },
  description: 'Visual dashboard for spwn — the control plane for AI agents',
  icons: [{ rel: 'icon', url: '/favicon.svg', type: 'image/svg+xml' }],
  openGraph: {
    title: 'Observatory · spwn',
    description: 'Visual dashboard for spwn — the control plane for AI agents',
    type: 'website',
    siteName: 'spwn Observatory',
  },
  metadataBase: new URL('https://spwn.sh'),
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html
      lang="en"
      className={`${geistSans.variable} ${geistMono.variable} ${spaceGrotesk.variable} h-full`}
      suppressHydrationWarning
    >
      <head>
        <script dangerouslySetInnerHTML={{ __html: `try{if(window.__TAURI_INTERNALS__||window.__TAURI__)document.documentElement.classList.add('tauri')}catch(e){}` }} />
      </head>
      <body className="h-svh overflow-hidden relative">
        <div className="flex flex-col h-full">
          <ThemeProvider>
            <TooltipProvider>
              <ToastProvider>
                <Aurora />
                <Stars />
                <div className="relative z-10 flex-1 min-h-0 h-full">
                  <AppShell>{children}</AppShell>
                  <CommandPalette />
                </div>
              </ToastProvider>
            </TooltipProvider>
          </ThemeProvider>
        </div>
      </body>
    </html>
  );
}
