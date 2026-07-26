import { Skeleton } from "@/components/ui/skeleton";

// Route-level Suspense boundary: without this, clicking into /dashboard
// left the previous route's content frozen on screen until the new
// segment's data arrived, since there was no interim UI for the App Router
// to show during the transition itself. Mirrors PuzzleGrid's own loading
// skeleton so there's no visible swap once real data replaces this.
export default function DashboardLoading() {
  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div className="flex flex-col gap-2">
          <Skeleton className="h-8 w-56" />
          <Skeleton className="h-4 w-40" />
        </div>
        <Skeleton className="h-10 w-36" />
      </div>
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 3xl:grid-cols-4">
        {Array.from({ length: 6 }).map((_, i) => (
          <Skeleton key={i} className="h-32 w-full" />
        ))}
      </div>
    </div>
  );
}
