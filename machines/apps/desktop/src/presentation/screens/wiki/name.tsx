import { useRead } from '../../dependencies/dependencies';

// value: a wiki page named in the toolbar as the engine serves it: its seat, then its title.
export function WikiPageName({ id }: { id: string }) {
    const reading = useRead('wiki.read', { Page: id }, undefined, { placeholder: true });
    if (reading.kind !== 'read') {
        return id;
    }
    const { Page, Seats } = reading.out;
    const seat = (Seats ?? []).find((each) => each.Active)?.Name ?? '';
    return seat === '' ? Page.Word : `${seat} › ${Page.Word}`;
}
