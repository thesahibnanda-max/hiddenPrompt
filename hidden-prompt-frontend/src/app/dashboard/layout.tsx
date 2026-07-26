import { SiteHeader } from "@/components/chrome/SiteHeader";

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex min-h-screen flex-col">
      <SiteHeader />
      <main className="mx-auto w-full max-w-6xl flex-1 px-4 py-8 sm:px-6 lg:px-8 3xl:px-24 3xl:py-14 tv:px-40">
        {children}
      </main>
    </div>
  );
}
