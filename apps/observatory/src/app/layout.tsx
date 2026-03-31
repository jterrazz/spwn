import type { Metadata } from "next";
import { Geist, Geist_Mono, Space_Grotesk } from "next/font/google";
import "./globals.css";
import { Aurora } from "@/components/aurora";
import { Stars } from "@/components/stars";
import { ThemeProvider } from "@/components/theme-provider";
import { TooltipProvider } from "@/components/ui/tooltip";

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
  title: "spwn observatory",
  description: "Visual dashboard for your universe",
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
      <body className="min-h-full relative overflow-x-hidden overflow-y-auto">
        <ThemeProvider>
          <TooltipProvider>
            <Aurora />
            <Stars />
            <div className="relative z-10 min-h-screen">
              {children}
            </div>
          </TooltipProvider>
        </ThemeProvider>
      </body>
    </html>
  );
}
