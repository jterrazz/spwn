/** A ref callback that focuses the element it is attached to, once mounted: the first field of a step, or of a dialog. */
export function focusOnMount(element: HTMLElement | null): void {
    element?.focus();
}
