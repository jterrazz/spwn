import { Skeleton } from '@/components/ui/skeleton';

export default function ArchitectLoading() {
    return (
        <div className="px-6 pt-8 pb-16 space-y-8">
            <div>
                <Skeleton className="h-8 w-40" />
                <Skeleton className="h-3 w-56 mt-2" />
            </div>
            <div className="space-y-4">
                <Skeleton className="h-24 w-full rounded-xl" />
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    <Skeleton className="h-40 rounded-xl" />
                    <Skeleton className="h-40 rounded-xl" />
                </div>
            </div>
        </div>
    );
}
