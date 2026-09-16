import { Skeleton } from '@/components/ui/skeleton';

export default function HomeLoading() {
    return (
        <div className="flex min-h-screen flex-col">
            <div className="flex items-start justify-between px-6 pt-8">
                <div>
                    <Skeleton className="h-8 w-32" />
                    <Skeleton className="mt-2 h-3 w-24" />
                </div>
                <Skeleton className="h-10 w-36 rounded-xl" />
            </div>
            <main className="flex flex-1 items-center justify-center py-16">
                <div className="flex items-center gap-12 md:gap-20">
                    {[1, 2, 3].map((i) => (
                        <div className="flex flex-col items-center gap-4" key={i}>
                            <Skeleton className="h-24 w-24 rounded-full" />
                            <Skeleton className="h-4 w-16" />
                            <Skeleton className="h-3 w-24" />
                        </div>
                    ))}
                </div>
            </main>
        </div>
    );
}
