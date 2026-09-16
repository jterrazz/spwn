import { Skeleton } from '@/components/ui/skeleton';

export default function AgentLoading() {
    return (
        <div className="space-y-8 px-6 pt-8 pb-16">
            <div className="flex items-center gap-4">
                <Skeleton className="h-12 w-12 rounded-full" />
                <div>
                    <Skeleton className="h-6 w-32" />
                    <Skeleton className="mt-2 h-3 w-48" />
                </div>
            </div>
            <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
                {[1, 2, 3].map((i) => (
                    <Skeleton className="h-28 rounded-xl" key={i} />
                ))}
            </div>
            <div className="space-y-3">
                <Skeleton className="h-5 w-24" />
                {[1, 2, 3, 4].map((i) => (
                    <Skeleton className="h-10 w-full rounded-lg" key={i} />
                ))}
            </div>
        </div>
    );
}
