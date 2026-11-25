import type { Request, Target } from '@jterrazz/station-api';
import { button, enabled, region, within } from '@jterrazz/test';

import { pageOver } from '../../../../composition/__tests__/page-over';
import { isRecord } from '../../../../domain/request-key';
import { documentPresence } from '../../../../infrastructure/outbound/document-presence/document-presence';
import { Client } from '../../../../infrastructure/outbound/engine-link/client';
import { stagedReplay } from '../../../../infrastructure/outbound/engine-replay/replay';
import type { Recording, Stage } from '../../../../infrastructure/outbound/engine-replay/replay';
import { NO_OPENER } from '../../../../infrastructure/outbound/opener-tauri/tauri-opener';
import { EngineProvider } from '../../../dependencies/dependencies';
import { Window } from '../../../navigation/window';
import { PAGES } from '../../../screens/registry';

export type Free = { Holder: string; Kind: string; Name: string; Bytes: number };

export const SUBJECT = 'base-ubuntu@2026.10.20, old-vm';

export const NAMES = ['base-ubuntu@2026.10.20', 'old-vm'];

export const THIS_MAC = region('macbook-pro');
export const MINI = region('mac-mini');
export const disk = within(THIS_MAC, region('Disk'));
export const freed = within(THIS_MAC, region('Can be freed'));
export const kept = within(THIS_MAC, region('Kept'));
export const done = within(THIS_MAC, region('Done here'));
export const freeIt = enabled(button('Free'));
export const freeStation = enabled(button('Free 12 GB'));
export const freeCaches = enabled(button('Free 34.8 GB'));

export function staged(recording: Recording): Stage {
    return stagedReplay(recording, {
        hello: { pid: 41, started_at: '2026-11-16T09:00:00Z', version: 'dev' },
        kind: 'open',
        refused: undefined,
        socket: 'replay',
    });
}

export function storageOn(stage: Stage) {
    const dependencies = {
        engine: new Client(stage.transport),
        opener: NO_OPENER,
        presence: documentPresence(),
    };
    return (
        <EngineProvider dependencies={dependencies}>
            <Window initial={{ page: 'storage' }} pages={PAGES} />
        </EngineProvider>
    );
}

export function storageOver(recording: Recording) {
    return pageOver(recording, { page: 'storage' });
}

export function freesAsked(stage: Stage): readonly unknown[] {
    return stage
        .asked()
        .flatMap((request: Request) =>
            'call' in request && request.call === 'disk.free-unused' ? [request.in] : [],
        );
}

export function readsAsked(stage: Stage, read: string): number {
    return stage
        .asked()
        .filter(
            (request) =>
                ('call' in request && request.call === read) ||
                ('watch' in request && request.watch === read),
        ).length;
}

export function answeredBy(recording: Recording, other: Recording): Recording {
    return { ...recording, gestures: other.gestures ?? [] };
}

export function withStation(
    recording: Recording,
    change: (station: Record<string, unknown>) => unknown,
) {
    return {
        ...recording,
        exchanges: recording.exchanges.map((exchange) =>
            exchange.request.watch === 'disk.read'
                ? {
                      ...exchange,
                      answers: exchange.answers.map((answer: unknown) =>
                          isRecord(answer) && isRecord(answer.out) && isRecord(answer.out.Station)
                              ? {
                                    ...answer,
                                    out: { ...answer.out, Station: change(answer.out.Station) },
                                }
                              : answer,
                      ),
                  }
                : exchange,
        ),
    };
}

export function refusal(subject: string, typed: string) {
    return {
        v: 2,
        error: {
            type: 'needs-confirmation',
            code: 'confirm.yes-needed',
            subject,
            detail: { level: 'yes-no', name: subject, typed },
            message: `refusing without --yes: ${subject}`,
            hint: 're-run with --yes',
        },
    };
}

export function gestureAnswering(request: Record<string, unknown>, answer: unknown) {
    return { request: { call: 'disk.free-unused', ...request }, answers: [answer], after: [] };
}

export function freeing(recording: Recording, frees: readonly Free[]) {
    return withStation(recording, (station) => ({
        ...station,
        CanFree: {
            Bytes: frees.reduce((sum, one) => sum + one.Bytes, 0),
            Call: 'disk.free-unused',
            Frees: frees,
            Sentence: `${frees.length} unused`,
        },
    }));
}

export function withMini(recording: Recording, frees: readonly Free[]): Recording {
    const mac = recording.exchanges.find(
        (exchange) =>
            exchange.request.watch === 'disk.read' && exchange.request.target === undefined,
    );
    if (mac === undefined) {
        throw new Error('two-macs holds the disk of this Mac.');
    }
    const [read] = freeing({ ...recording, exchanges: [mac] }, frees).exchanges;
    if (read === undefined) {
        throw new Error('freeing keeps its exchange.');
    }
    const mini = { ...read, request: { ...read.request, target: { name: 'mac-mini' } } };
    return { ...recording, exchanges: [...recording.exchanges, mini] };
}

export const HELLO_AGAIN = {
    pid: 42,
    started_at: '2026-11-16T09:05:00Z',
    version: 'dev',
};

export function sentTo(stage: Stage): readonly { names: unknown; target: Target | undefined }[] {
    return stage.asked().flatMap((request: Request) =>
        'call' in request && request.call === 'disk.free-unused'
            ? [
                  {
                      names: isRecord(request.in) ? request.in.Names : undefined,
                      target: request.target,
                  },
              ]
            : [],
    );
}

export function eachAsking(recording: Recording, own: string, other: string): Recording {
    const withOther = withMini(recording, [
        { Bytes: 2_000_000_000, Holder: 'images', Kind: 'image', Name: other },
    ]);
    return {
        ...withOther,
        gestures: [
            ...(withOther.gestures ?? []),
            gestureAnswering({ in: { Names: [own] } }, refusal(own, '')),
            gestureAnswering(
                { in: { Names: [other] }, target: { name: 'mac-mini' } },
                refusal(other, ''),
            ),
        ],
    };
}

export function narrowedTo(recording: Recording, first: string, typed: string): Recording {
    return {
        ...freeing(recording, [
            { Bytes: 3_000_000_000, Holder: 'images', Kind: 'image', Name: first },
        ]),
        gestures: [
            gestureAnswering({ in: { Confirm: typed, Names: [first] } }, refusal(first, typed)),
        ],
    };
}

function broken(Name: string, Reason: string) {
    return { Detail: '', Erase: false, Freed: 0, Hint: '', Name, Reason, Result: 'broken' };
}

export function everyOutcomeBroken(recording: Recording): Recording {
    return {
        ...recording,
        gestures: (recording.gestures ?? []).map((gesture) =>
            isRecord(gesture.request.in) && 'Confirm' in gesture.request.in
                ? {
                      ...gesture,
                      after: [],
                      answers: [
                          {
                              v: 2,
                              out: [
                                  broken('base-ubuntu@2026.10.20', 'the image is in use'),
                                  broken('~/.tart', 'tart is running'),
                              ],
                          },
                      ],
                  }
                : gesture,
        ),
    };
}
