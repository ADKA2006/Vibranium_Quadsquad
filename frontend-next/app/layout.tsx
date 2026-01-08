import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import "./globals.css";
import { AuthProvider } from "@/lib/auth-context";
import { BlockedCountriesProvider } from "@/lib/blocked-countries-context";
import { WebSocketProvider } from "@/lib/websocket-context";

const geistSans = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: "Predictive Liquidity Mesh",
  description: "Real-time mesh visualization with admin controls",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" className="dark">
      <body
        className={`${geistSans.variable} ${geistMono.variable} antialiased bg-gradient-to-br from-slate-950 via-slate-900 to-indigo-950 min-h-screen`}
      >
        <AuthProvider>
          <BlockedCountriesProvider>
            <WebSocketProvider>
              {children}
            </WebSocketProvider>
          </BlockedCountriesProvider>
        </AuthProvider>
      </body>
    </html>
  );
}
