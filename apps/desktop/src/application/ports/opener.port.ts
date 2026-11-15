import type { EngineError } from '../../domain/engine-error';

export type Opens = 'terminal' | 'url';

export type Opening = { readonly opens: Opens; readonly value: string };

export type Opened = { kind: 'opened' } | { kind: 'refused'; error: EngineError };

export type OpenerPort = (opening: Opening) => Promise<Opened>;
