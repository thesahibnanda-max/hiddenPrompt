import type { Metadata } from "next";
import { fontDisplay, fontBody } from "@/styles/fonts";
import { SynthwaveBackground } from "@/components/chrome/SynthwaveBackground";
import { MusicSystem } from "@/components/chrome/MusicSystem";
import { GithubSourceLink } from "@/components/chrome/GithubSourceLink";
import { PageTransition } from "@/components/chrome/PageTransition";
import { AuthGate } from "@/components/chrome/AuthGate";
import { Toaster } from "@/components/ui/toaster";
import "./globals.css";

export const metadata: Metadata = {
  title: "Hidden Prompt — Reverse Prompt Intelligence",
  description:
    "The AI gives you a one-line answer. You reverse-engineer the question. Scored by robots who are not impressed.",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" className={`${fontDisplay.variable} ${fontBody.variable}`}>
      {/* pb-96 reserves room below every page's content on mobile so the
          fixed-position music panel (bottom-4, ~360-400px tall in its
          mini-player/expanded states) never sits on top of the last thing
          on a short page - back to 0 at sm: where the panel returns to its
          compact, fixed-width corner widget and pages are generally tall
          enough already. */}
      <body className="pb-96 sm:pb-0">
        <SynthwaveBackground />
        <MusicSystem />
        <GithubSourceLink />
        <Toaster />
        <AuthGate>
          <PageTransition>{children}</PageTransition>
        </AuthGate>
      </body>
    </html>
  );
}
