import { Orbitron, Space_Grotesk } from "next/font/google";

export const fontDisplay = Orbitron({
  subsets: ["latin"],
  weight: ["700", "900"],
  variable: "--font-display",
});

export const fontBody = Space_Grotesk({
  subsets: ["latin"],
  weight: ["400", "500", "600", "700"],
  variable: "--font-body",
});
