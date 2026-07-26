import { AuthTabs } from "@/components/auth/AuthTabs";
import { HowToPlaySection } from "@/components/how-to-play/HowToPlaySection";

export default function LandingPage() {
  return (
    <div className="flex min-h-screen flex-col items-center gap-14 px-4 py-12 sm:px-6 lg:px-8 3xl:px-24 3xl:py-20 tv:px-40">
      <AuthTabs />
      <HowToPlaySection />
    </div>
  );
}
