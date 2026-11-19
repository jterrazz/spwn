import { CORE_FOLDER } from '../../../domain/core';
import { Mono } from './core-parts';
import type { Row } from './core-parts';

export const PUTS: readonly Row[] = [
    {
        key: 'station',
        mono: true,
        place: CORE_FOLDER,
        value: 'The engine and its CLI: the machines',
    },
    { key: 'spwn', mono: true, place: CORE_FOLDER, value: 'The agents' },
    {
        key: 'Links',
        place: CORE_FOLDER,
        value: (
            <>
                <Mono>spwn-machines</Mono>, <Mono>spwn-agents</Mono>
            </>
        ),
    },
    {
        key: 'Shell profile',
        place: '~/.zshrc on zsh',
        value: (
            <>
                One PATH line and two aliases, under <Mono># Added by spwn installer</Mono>
            </>
        ),
    },
];

export const PROMISES = [
    'No admin password',
    'Nothing is installed unless every checksum matches',
    'The engine is not started yet',
];

export const ON_DISK: readonly Row[] = [
    { key: 'station', mono: true, place: CORE_FOLDER, value: 'The engine and its CLI' },
    { key: 'spwn', mono: true, place: CORE_FOLDER, value: 'The agents' },
];

export const STARTING: readonly Row[] = [
    { key: 'Names this Mac', value: "From the Mac's own name" },
    { key: 'Runs at login', value: 'From now on the engine starts with the Mac' },
    { key: 'Asks once', value: 'macOS asks to let station reach the local network' },
];
