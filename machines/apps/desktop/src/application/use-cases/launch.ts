import type { LaunchPort } from '../ports/launch.port';
import type { Reading } from './entry';
import type { Reach } from './reach';

export const LAUNCH_CAP = 800;

// value: machines.list's first out or refusal, or no engine answering, the core's panel then being the page.
export function holdsData(reach: Reach, machines: Reading<unknown>): boolean {
    return reach.kind === 'unanswered' || machines.kind !== 'reading';
}

export class Launch {
    private readonly port: LaunchPort;
    private drawnYet = false;
    private facedYet = false;
    private shownYet = false;

    constructor(port: LaunchPort) {
        this.port = port;
    }

    begin(): void {
        this.port.after(Math.max(0, LAUNCH_CAP - this.port.since()), () => {
            this.reveal();
        });
        void this.faced();
    }

    readonly drawn = (): void => {
        this.drawnYet = true;
        this.settle();
    };

    private async faced(): Promise<void> {
        await this.port.faces();
        this.facedYet = true;
        this.settle();
    }

    private settle(): void {
        if (this.drawnYet && this.facedYet) {
            this.reveal();
        }
    }

    private reveal(): void {
        if (this.shownYet) {
            return;
        }
        this.shownYet = true;
        void this.port.show();
    }
}
