import { Skeleton } from '@/components/ui/skeleton';

export default function ProvidersLoading() {
    return (
        <div className="space-y-6 px-6 pt-6 pb-16">
            <div>
                <Skeleton className="h-8 w-48" />
                <Skeleton className="mt-2 h-3 w-64" />
            </div>
            <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
                {[1, 2, 3].map((i) => (
                    <Skeleton className="h-64 rounded-xl" key={i} />
                ))}
            </div>
        </div>
    );
}
