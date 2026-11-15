import type { ActivityReport } from '@jterrazz/station-api';
import type { ReactNode } from 'react';

import { duration, sentence } from '../../domain/format';
import { between, firstUnitOf } from '../../domain/when';
import { Facts, MissingMark, Panel } from '../../presentation/ui/index';
import { BATTERY_WORDS, THERMAL_WORDS } from '../../presentation/words/words';

function upFor(report: ActivityReport): ReactNode {
    const { boot, at } = report.last.sample;
    const up = between(boot, at);
    if (up === undefined) {
        return <MissingMark />;
    }
    const { value, unit } = firstUnitOf(duration(up));
    return `${value} ${unit}`;
}

function batteryOf(report: ActivityReport): ReactNode {
    const { battery } = report.last.sample;
    if (!battery) {
        return <MissingMark />;
    }
    return sentence([
        `${String(battery.percent)} %`,
        battery.charging ? BATTERY_WORDS.charging : undefined,
        battery.on_battery ? BATTERY_WORDS.onBattery : undefined,
    ]);
}

function healthOf(report: ActivityReport): ReactNode {
    const { battery } = report.last.sample;
    const health = sentence([
        battery?.health === undefined ? undefined : `${String(battery.health)} %`,
        battery?.cycles === undefined ? undefined : `${String(battery.cycles)} cycles`,
    ]);
    return health === '' ? <MissingMark /> : health;
}

export function MacFacts({ report, title }: { report: ActivityReport; title: string }) {
    const { sample } = report.last;
    return (
        <Panel padded title={title}>
            <Facts
                facts={[
                    {
                        label: 'Heat',
                        value: THERMAL_WORDS[String(sample.thermal)] ?? <MissingMark />,
                    },
                    { label: 'Battery', value: batteryOf(report) },
                    { label: 'Battery health', value: healthOf(report) },
                    { label: 'Up for', value: upFor(report) },
                ]}
            />
        </Panel>
    );
}
