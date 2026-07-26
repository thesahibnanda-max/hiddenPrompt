"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { motion } from "framer-motion";
import { toast } from "sonner";
import { PartyPopper } from "lucide-react";
import { Button } from "@/components/ui/button";
import { initiateVerification, decideVerification } from "@/lib/api/endpoints";
import { ApiError, RateLimitError } from "@/lib/api/errors";
import { useAuthStore } from "@/stores/auth-store";
import { SendCodeStep } from "./SendCodeStep";
import { OtpStep } from "./OtpStep";

type Step = "send" | "otp" | "done";

export function VerifyGate() {
  const router = useRouter();
  const setVerified = useAuthStore((s) => s.setVerified);

  const [step, setStep] = useState<Step>("send");
  const [otp, setOtp] = useState("");
  const [loading, setLoading] = useState(false);
  const [resendLoading, setResendLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [otpError, setOtpError] = useState<string | null>(null);
  const [shake, setShake] = useState(false);
  const [rateLimitSeconds, setRateLimitSeconds] = useState<number | null>(null);
  const [alreadyVerifiedHint, setAlreadyVerifiedHint] = useState<string | null>(null);

  const send = async () => {
    setError(null);
    setAlreadyVerifiedHint(null);
    setLoading(true);
    try {
      await initiateVerification();
      toast.success("Check your inbox — we sent you a secret code.");
      setStep("otp");
    } catch (err) {
      if (err instanceof RateLimitError) {
        setRateLimitSeconds(err.retryAfterSeconds);
      } else if (err instanceof ApiError) {
        if (err.statusCode === 409) {
          setAlreadyVerifiedHint("Plot twist: you're already verified. Skip ahead.");
          setVerified(true);
        } else {
          setError(err.showAsIs ? err.body.error_message : "Couldn't send that. Try again in a bit.");
        }
      } else {
        setError("Couldn't send that. Try again in a bit.");
      }
    } finally {
      setLoading(false);
    }
  };

  const resend = async () => {
    setResendLoading(true);
    setOtpError(null);
    try {
      await initiateVerification();
      toast.success("New code on its way.");
    } catch (err) {
      if (err instanceof RateLimitError) {
        setRateLimitSeconds(err.retryAfterSeconds);
      } else if (err instanceof ApiError) {
        setOtpError(err.showAsIs ? err.body.error_message : "Couldn't resend that. Try again in a bit.");
      }
    } finally {
      setResendLoading(false);
    }
  };

  const submitOtp = async () => {
    setOtpError(null);
    setLoading(true);
    try {
      await decideVerification({ otp });
      setVerified(true);
      setStep("done");
      setTimeout(() => router.replace("/dashboard"), 1400);
    } catch (err) {
      if (err instanceof RateLimitError) {
        setRateLimitSeconds(err.retryAfterSeconds);
      } else if (err instanceof ApiError) {
        setOtpError(
          err.showAsIs
            ? err.body.error_message
            : "Nope. Either that code's wrong or it already expired.",
        );
        setShake(true);
        setTimeout(() => setShake(false), 400);
      }
    } finally {
      setLoading(false);
    }
  };

  if (step === "done") {
    return (
      <motion.div
        initial={{ opacity: 0, scale: 0.9 }}
        animate={{ opacity: 1, scale: 1 }}
        className="flex flex-col items-center gap-4 text-center"
      >
        <PartyPopper className="text-neon-cyan" size={40} />
        <h2 className="chrome-text text-2xl font-black">VERIFIED!</h2>
        <p className="text-sm text-chrome-mid">Taking you to your puzzles...</p>
      </motion.div>
    );
  }

  if (step === "otp") {
    return (
      <OtpStep
        otp={otp}
        onOtpChange={setOtp}
        onSubmit={submitOtp}
        onResend={resend}
        loading={loading}
        resendLoading={resendLoading}
        error={otpError}
        shake={shake}
        rateLimitSeconds={rateLimitSeconds}
        onRateLimitDone={() => setRateLimitSeconds(null)}
      />
    );
  }

  if (alreadyVerifiedHint) {
    return (
      <div className="flex flex-col items-center gap-4 text-center">
        <p className="text-sm text-neon-cyan">{alreadyVerifiedHint}</p>
        <Button onClick={() => router.replace("/dashboard")} size="lg">
          Take me to my puzzles
        </Button>
      </div>
    );
  }

  return (
    <SendCodeStep
      onSend={send}
      loading={loading}
      error={error}
      rateLimitSeconds={rateLimitSeconds}
      onRateLimitDone={() => setRateLimitSeconds(null)}
      alreadyVerifiedHint={alreadyVerifiedHint}
    />
  );
}
