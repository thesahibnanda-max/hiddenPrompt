"use client";

import Link from "next/link";
import { LogOut, Sparkles } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useAuthStore } from "@/stores/auth-store";

export function SiteHeader({ rightSlot }: { rightSlot?: React.ReactNode }) {
  const logout = useAuthStore((s) => s.logout);

  return (
    <header className="sticky top-0 z-30 border-b border-white/10 bg-void/80 backdrop-blur-md">
      <div className="mx-auto flex max-w-6xl items-center justify-between px-4 py-3 sm:px-6 lg:px-8 3xl:px-24 3xl:py-5 tv:px-40">
        <Link href="/dashboard" className="flex items-center gap-2">
          <Sparkles className="text-neon-cyan" size={22} />
          <span className="chrome-text text-lg font-black tracking-wide 3xl:text-2xl">
            HIDDEN PROMPT
          </span>
        </Link>
        <div className="flex items-center gap-3">
          {rightSlot}
          <Button
            variant="ghost"
            size="sm"
            onClick={logout}
            aria-label="Log out"
          >
            <LogOut size={16} />
            <span className="hidden sm:inline">Log out</span>
          </Button>
        </div>
      </div>
    </header>
  );
}
