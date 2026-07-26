import Link from "next/link";
import { Button } from "@/components/ui/button";

export default function NotFound() {
  return (
    <div className="flex min-h-screen flex-col items-center justify-center gap-4 px-4 text-center">
      <h1 className="chrome-text text-6xl font-black">404</h1>
      <p className="max-w-sm text-chrome-mid">
        This page doesn&apos;t exist. Much like a good guess on your first try.
      </p>
      <Button asChild>
        <Link href="/">Take me home</Link>
      </Button>
    </div>
  );
}
