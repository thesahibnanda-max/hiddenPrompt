import { Check, X } from "lucide-react";
import { checkPasswordRules } from "@/lib/utils/password";
import { cn } from "@/lib/utils/cn";

export function PasswordRules({ password }: { password: string }) {
  const rules = checkPasswordRules(password);
  return (
    <ul className="grid grid-cols-1 gap-1 text-xs sm:grid-cols-2">
      {rules.map((rule) => (
        <li
          key={rule.key}
          className={cn(
            "flex items-center gap-1.5",
            rule.passed ? "text-neon-cyan" : "text-chrome-mid",
          )}
        >
          {rule.passed ? <Check size={13} /> : <X size={13} className="opacity-40" />}
          {rule.label}
        </li>
      ))}
    </ul>
  );
}
