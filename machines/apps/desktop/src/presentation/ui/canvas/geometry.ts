export type Size = readonly [width: number, height: number];

export type Geometry = {
    cards: Readonly<Record<number, Size>>;
    otherCard: Size;
    emptyCard: Size;
    pad: number;
    gap: number;
    head: number;
    headByDepth: Readonly<Record<number, number>>;
    minOpen: Readonly<Record<number, number>>;
    otherMinOpen: number;
    regionColumns: number;
    leafDepth: number;
    row: { height: number; pitch: number; after: number; pillGap: number; dotGap: number };
    pill: {
        height: number;
        lead: number;
        tail: number;
        labelCap: number;
        marksLead: number;
        markPitch: number;
        marksCap: number;
    };
    dot: {
        size: number;
        chipDepth: number;
        chipLead: number;
        chipPitch: number;
        chipCap: number;
        chipTail: number;
        chipHeight: number;
    };
    layout: {
        maxWidth: number;
        sideMargin: number;
        gutter: number;
        seatGap: number;
        bandGap: number;
        bandGapOpen: number;
        guideLift: number;
        bandFloor: number;
    };
    camera: {
        fitHeight: number;
        fitWidth: number;
        topOpen: number;
        topRoot: number;
        clamp: number;
        visible: number;
    };
    neighbour: { clipWidth: number; clipHeight: number; ahead: number; across: number };
};

// value: the prototype's numbers (`SZ`, `MINOPEN`, `place`, `layout`, `camTarget`, `camClamp`, `ensureVisible` and
// value: the arrow keys' score of wiki-proto/spwn-wiki-proto-r1.html), the layout by meaning the owner chose
export const GEOMETRY: Geometry = {
    camera: { clamp: 120, fitHeight: 56, fitWidth: 48, topOpen: 70, topRoot: 36, visible: 24 },
    cards: { 2: [248, 118], 3: [236, 116] },
    dot: {
        chipCap: 10,
        chipDepth: 1,
        chipHeight: 24,
        chipLead: 31,
        chipPitch: 8,
        chipTail: 5,
        size: 20,
    },
    emptyCard: [248, 118],
    gap: 12,
    head: 46,
    headByDepth: { 2: 80 },
    layout: {
        bandFloor: 30,
        bandGap: 52,
        bandGapOpen: 40,
        guideLift: 20,
        gutter: 132,
        maxWidth: 1100,
        seatGap: 28,
        sideMargin: 120,
    },
    leafDepth: 4,
    minOpen: { 1: 300, 2: 560, 3: 420 },
    neighbour: { across: 2.2, ahead: 4, clipHeight: 46, clipWidth: 240 },
    otherCard: [222, 108],
    otherMinOpen: 300,
    pad: 16,
    pill: {
        height: 30,
        labelCap: 190,
        lead: 33,
        markPitch: 8,
        marksCap: 11,
        marksLead: 9,
        tail: 12,
    },
    regionColumns: 3,
    row: { after: 14, dotGap: 6, height: 30, pillGap: 8, pitch: 38 },
};
