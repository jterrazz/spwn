export type Page = Pick<Document, 'visibilityState'> & {
    addEventListener: (type: 'visibilitychange', listener: () => void) => void;
};

export type Root = Pick<HTMLElement, 'dataset'>;

export function markVisibility(root: Root, page: Page): void {
    const mark = (): void => {
        root.dataset.visibility = page.visibilityState === 'hidden' ? 'hidden' : 'visible';
    };
    mark();
    page.addEventListener('visibilitychange', mark);
}
