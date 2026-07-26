import { Lock } from "lucide-react";

export function AlreadySolvedBanner() {
  return (
    <div className="flex items-center gap-2 rounded-md border border-neon-cyan/30 bg-neon-cyan/5 px-4 py-3 text-sm text-neon-cyan">
      <Lock size={16} className="shrink-0" />
      <span>Already cracked — someone (you, five seconds ago) beat this one.</span>
    </div>
  );
}
