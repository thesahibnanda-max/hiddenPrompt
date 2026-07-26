"use client";

import { useState } from "react";
import { motion } from "framer-motion";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { RateLimitBanner } from "@/components/feedback/RateLimitBanner";
import { PasswordRules } from "./PasswordRules";
import { signUp } from "@/lib/api/endpoints";
import { ApiError, RateLimitError } from "@/lib/api/errors";
import { useAuthStore } from "@/stores/auth-store";
import { isPasswordValid } from "@/lib/utils/password";

export function SignupForm() {
  const setEmail = useAuthStore((s) => s.setEmail);
  const setVerified = useAuthStore((s) => s.setVerified);

  const [email, setEmailField] = useState("");
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [touched, setTouched] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [rateLimit, setRateLimit] = useState<RateLimitError | null>(null);
  const [loading, setLoading] = useState(false);

  const mismatch = touched && confirm.length > 0 && password !== confirm;

  const onSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setTouched(true);
    setError(null);
    setRateLimit(null);

    if (password !== confirm) {
      setError("Those two passwords don't match. Try that again.");
      return;
    }

    setLoading(true);
    try {
      await signUp({ email, password });
      // Navigation is owned entirely by (auth)/layout.tsx, which reacts to
      // the token/verified state this sets — see the redirect-race note
      // there for why this form must not also call router.replace.
      setEmail(email);
      setVerified(false); // brand-new accounts always start unverified — authoritative here
    } catch (err) {
      if (err instanceof RateLimitError) {
        setRateLimit(err);
      } else if (err instanceof ApiError) {
        setError(err.showAsIs ? err.body.error_message : "Something went wrong creating your account. Try again?");
      } else {
        setError("Something went wrong creating your account. Try again?");
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
        <Label htmlFor="signup-email">Email</Label>
        <Input
          id="signup-email"
          type="email"
          autoComplete="email"
          required
          value={email}
          onChange={(e) => setEmailField(e.target.value)}
          placeholder="you@example.com"
        />
      </div>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="signup-password">Password</Label>
        <Input
          id="signup-password"
          type="password"
          autoComplete="new-password"
          required
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          onBlur={() => setTouched(true)}
          placeholder="••••••••"
        />
        <PasswordRules password={password} />
      </div>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="signup-confirm">Confirm password</Label>
        <Input
          id="signup-confirm"
          type="password"
          autoComplete="new-password"
          required
          value={confirm}
          onChange={(e) => setConfirm(e.target.value)}
          onBlur={() => setTouched(true)}
          placeholder="••••••••"
        />
        {mismatch && <p className="text-xs text-destructive">Passwords don&apos;t match yet.</p>}
      </div>

      {error && <p className="text-sm text-destructive">{error}</p>}
      {rateLimit && (
        <RateLimitBanner
          retryAfterSeconds={rateLimit.retryAfterSeconds}
          onDone={() => setRateLimit(null)}
        />
      )}

      <Button
        type="submit"
        variant="neon"
        size="lg"
        disabled={loading || !!rateLimit || !isPasswordValid(password)}
        className="mt-2"
      >
        {loading ? "Creating account..." : "Sign Up"}
      </Button>
    </motion.form>
  );
}
