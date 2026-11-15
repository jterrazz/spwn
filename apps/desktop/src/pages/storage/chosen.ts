import type { ImagesReport, MachinesMacView } from '@jterrazz/station-api';

export function chosenMac(
    macs: readonly Pick<MachinesMacView, 'Name' | 'ThisMac'>[],
    chosen: string | undefined,
): string | undefined {
    if (chosen !== undefined && macs.some((mac) => mac.Name === chosen)) {
        return chosen;
    }
    return (macs.find((mac) => mac.ThisMac) ?? macs[0])?.Name;
}

export function imagesOn(images: readonly ImagesReport[], mac: string): readonly ImagesReport[] {
    return images.filter((image) => image.Mac === mac);
}
