import { SiteHeader } from "@/components/chrome/SiteHeader";
import { Skeleton } from "@/components/ui/skeleton";

// Route-level Suspense boundary: without this, tapping a puzzle from the
// dashboard left the dashboard's own content frozen on screen until this
// route's data arrived, before the page's own component-level skeleton
// (usePuzzleDetail's `loading` state) ever got a chance to render. This
// route has no shared layout.tsx of its own, so SiteHeader is reproduced
// here to match page.tsx's structure exactly.
export default function PuzzleDetailLoading() {
  return (
    <div className="flex min-h-screen flex-col">
      <SiteHeader />
      <main className="mx-auto flex w-full max-w-3xl flex-1 flex-col gap-5 px-4 py-6 sm:px-6 lg:px-8 3xl:max-w-4xl 3xl:px-24 3xl:py-10 tv:px-40">
        <div className="flex flex-col gap-4">
          <Skeleton className="h-10 w-2/3" />
          <Skeleton className="h-24 w-full" />
          <Skeleton className="h-24 w-3/4" />
        </div>
      </main>
    </div>
  );
}
