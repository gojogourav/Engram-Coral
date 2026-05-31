import type { Metadata } from "next";
import { Inter } from "next/font/google";
import ClientLayout from "./ClientLayout";
import "./globals.css";

const inter = Inter({ subsets: ["latin"] });

export const metadata: Metadata = {
  title: "Engram SRE Command Center",
  description: "AI-Powered Infrastructure Self-Healing",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en">
      <body className={inter.className}>
        <ClientLayout>{children}</ClientLayout>
      </body>
    </html>
  );
}
