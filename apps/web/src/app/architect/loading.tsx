import { Skeleton } from '@/components/ui/skeleton';

export default function ArchitectLoading() {
    return (
        <div className="space-y-8 px-6 pt-8 pb-16">
            <div>
                <Skeleton className="h-8 w-40" />
                <Skeleton className="mt-2 h-3 w-56" />
            </div>
            <div className="space-y-4">
                <Skeleton className="h-24 w-full rounded-xl" />
                <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
                    <Skeleton className="h-40 rounded-xl" />
                    <Skeleton className="h-40 rounded-xl" />
                </div>
            </div>
        </div>
    );
}
