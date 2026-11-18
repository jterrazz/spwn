export type Place = 'mac' | 'overlaid' | 'plain';

export type Dress = {
    chrome: 'overlay' | undefined;
    material: 'vibrant' | undefined;
};

export type Viewport = {
    innerHeight: number;
    innerWidth: number;
    screen: { height: number; width: number };
};

export type View = Viewport & {
    addEventListener: (type: 'resize', listener: () => void) => void;
};

export function placeOf(inTauri: boolean, userAgent: string, overlayAsked: boolean): Place {
    if (inTauri) {
        return /Macintosh/u.test(userAgent) ? 'mac' : 'plain';
    }
    return overlayAsked ? 'overlaid' : 'plain';
}

export function fillsTheScreen(view: Viewport): boolean {
    return view.innerWidth === view.screen.width && view.innerHeight === view.screen.height;
}

export function dressOf(place: Place, fullscreen: boolean): Dress {
    const overlaid = place === 'overlaid' || (place === 'mac' && !fullscreen);
    return {
        chrome: overlaid ? 'overlay' : undefined,
        material: place === 'mac' ? 'vibrant' : undefined,
    };
}

export type Root = Pick<Element, 'removeAttribute' | 'setAttribute'>;

function put(root: Root, name: 'chrome' | 'material', value: string | undefined): void {
    if (value === undefined) {
        root.removeAttribute(`data-${name}`);
    } else {
        root.setAttribute(`data-${name}`, value);
    }
}

export function wearDress(root: Root, place: Place, view: View): void {
    const wear = (): void => {
        const dress = dressOf(place, place === 'mac' && fillsTheScreen(view));
        put(root, 'chrome', dress.chrome);
        put(root, 'material', dress.material);
    };
    wear();
    if (place === 'mac') {
        view.addEventListener('resize', wear);
    }
}
