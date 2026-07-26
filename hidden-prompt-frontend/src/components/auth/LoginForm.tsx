"use client";

import { useState } from "react";
import { motion } from "framer-motion";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { RateLimitBanner } from "@/components/feedback/RateLimitBanner";
import { logIn } from "@/lib/api/endpoints";
import { ApiError, RateLimitError } from "@/lib/api/errors";
import { useAuthStore } from "@/stores/auth-store";

export function LoginForm() {
  const setEmail = useAuthStore((s) => s.setEmail);
  const setVerified = useAuthStore((s) => s.setVerified);

  const [email, setEmailField] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [rateLimit, setRateLimit] = useState<RateLimitError | null>(null);
  const [loading, setLoading] = useState(false);

  const onSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setRateLimit(null);
    setLoading(true);
    try {
      const data = await logIn({ email, password });
      // Navigation is owned entirely by (auth)/layout.tsx, which reacts to
      // the token/verified state this sets — see the redirect-race note
      // there for why this form must not also call router.replace. Setting
      // `verified` from the response (rather than leaving it at whatever
      // stale value predates this login) is what actually closes the race
      // that could otherwise send an already-verified account to /verify.
      setEmail(email);
      setVerified(data.is_verified);
    } catch (err) {
      if (err instanceof RateLimitError) {
        setRateLimit(err);
      } else if (err instanceof ApiError) {
        setError(err.showAsIs ? err.body.error_message : "Something went wrong logging you in. Try again?");
      } else {
        setError("Something went wrong logging you in. Try again?");
      }
    } finally {
      setLoading(false);
    }
  };

  return (
    <motion.form
      onSubmit={onSubmit}
      initial={{ opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      className="flex flex-col gap-4"
    >
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="login-email">Email</Label>
        <Input
          id="login-email"
          type="email"
          autoComplete="email"
          required
          value={email}
          onChange={(e) => setEmailField(e.target.value)}
          placeholder="you@example.com"
        />
      </div>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="login-password">Password</Label>
        <Input
          id="login-password"
          type="password"
          autoComplete="current-password"
          required
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          placeholder="••••••••"
        />
      </div>

      {error && <p className="text-sm text-destructive">{error}</p>}
      {rateLimit && (
        <RateLimitBanner
          retryAfterSeconds={rateLimit.retryAfterSeconds}
          onDone={() => setRateLimit(null)}
        />
      )}

      <Button type="submit" variant="neon" size="lg" disabled={loading || !!rateLimit} className="mt-2">
        {loading ? "Logging in..." : "Log In"}
      </Button>
    </motion.form>
  );
}
