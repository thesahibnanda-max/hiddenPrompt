import Link from "next/link";
import { Frown } from "lucide-react";
import { Button } from "@/components/ui/button";

export function ErrorFallback({
  title = "Well, this is awkward.",
  message,
  href = "/dashboard",
  hrefLabel = "Back to puzzles",
}: {
  title?: string;
  message: string;
  href?: string;
  hrefLabel?: string;
}) {
  return (
    <div className="flex flex-col items-center gap-4 rounded-lg border border-white/10 bg-card p-10 text-center">
      <Frown className="text-neon-magenta" size={40} />
      <h2 className="chrome-text text-xl font-bold">{title}</h2>
      <p className="max-w-sm text-sm text-chrome-mid">{message}</p>
      <Button asChild variant="outline">
        <Link href={href}>{hrefLabel}</Link>
      </Button>
    </div>
  );
}
