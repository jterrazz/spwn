import { GeistMono } from 'geist/font/mono';
import { GeistSans } from 'geist/font/sans';
import type { Metadata } from 'next';
import Script from 'next/script';

// hazard: a `next/font/google` face is fetched while a page compiles, and one failed fetch broke every later page of a
// running `next dev`; each face ships in a package instead
import '@fontsource-variable/space-grotesk/index.css';
import './globals.css';
import { AppShell } from '../components/app-shell';
import { ArchitectChatWidget } from '../components/architect-chat-widget';
import { Aurora } from '../components/aurora';
import { CommandPalette } from '../components/command-palette';
import { ThemeProvider } from '../components/theme-provider';
import { ToastProvider } from '../components/toast-provider';
import { ArchitectChatProvider } from '../contexts/architect-chat-context';

export const metadata: Metadata = {
    title: {
        template: '%s · spwn',
        default: 'spwn',
    },
    description: 'Desktop app for spwn - manage AI agent worlds',
    icons: [{ rel: 'icon', url: '/favicon.svg', type: 'image/svg+xml' }],
    openGraph: {
        title: 'spwn',
        description: 'Desktop app for spwn - manage AI agent worlds',
        type: 'website',
        siteName: 'spwn',
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
            className={`${GeistSans.variable} ${GeistMono.variable} h-full`}
            lang="en"
            suppressHydrationWarning
        >
            <head>
                <Script id="tauri-class" strategy="beforeInteractive">
                    {`try{if(window.__TAURI_INTERNALS__||window.__TAURI__)document.documentElement.classList.add('tauri')}catch(e){}`}
                </Script>
            </head>
            <body className="relative h-svh overflow-hidden">
                <div className="flex h-full flex-col">
                    <ThemeProvider>
                        <ToastProvider>
                            <Aurora />
                            <div className="relative z-10 h-full min-h-0 flex-1">
                                <ArchitectChatProvider>
                                    <AppShell>{children}</AppShell>
                                    <ArchitectChatWidget />
                                </ArchitectChatProvider>
                                <CommandPalette />
                            </div>
                        </ToastProvider>
                    </ThemeProvider>
                </div>
            </body>
        </html>
    );
}
