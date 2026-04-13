import { Skeleton } from "@/components/ui/skeleton";

export default function HomeLoading() {
  return (
    <div className="flex flex-col min-h-screen">
      <div className="px-6 pt-8 flex items-start justify-between">
        <div>
          <Skeleton className="h-8 w-32" />
          <Skeleton className="h-3 w-24 mt-2" />
        </div>
        <Skeleton className="h-10 w-36 rounded-xl" />
      </div>
      <main className="flex-1 flex items-center justify-center py-16">
        <div className="flex items-center gap-12 md:gap-20">
          {[1, 2, 3].map((i) => (
            <div key={i} className="flex flex-col items-center gap-4">
              <Skeleton className="w-24 h-24 rounded-full" />
              <Skeleton className="h-4 w-16" />
              <Skeleton className="h-3 w-24" />
            </div>
          ))}
        </div>
      </main>
    </div>
  );
}
