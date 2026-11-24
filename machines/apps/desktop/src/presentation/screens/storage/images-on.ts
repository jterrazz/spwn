import type { ImagesReport } from '@jterrazz/station-api';

export function imagesOn<T extends Pick<ImagesReport, 'Mac'>>(
    images: readonly T[],
    mac: string,
): readonly T[] {
    return images.filter((image) => image.Mac === mac);
}
