import { Bar, Chart, ShareBar, Tile } from '../../design/index';
import type { SharePart } from '../../design/index';
import { Tiles } from '../page';
import { Section } from './section';

import styles from './gallery.module.css';

const MEMORY: readonly SharePart[] = [
    { identity: 'blue', key: 'a', size: 8 },
    { identity: 'purple', key: 'b', size: 10 },
    { key: 'free', size: 10 },
];

export function Measures() {
    return (
        <Section title="Tile, bar and chart">
            <div className={styles.stack}>
                <Tiles>
                    <Tile
                        foot="of 128 · room for 4 more gates"
                        label="Memory free"
                        unit="GB"
                        value={48}
                        viz={<ShareBar label="Memory" parts={MEMORY} />}
                    />
                    <Tile
                        foot="Peak 91 % · 14 min ago"
                        label="CPU"
                        unit="%"
                        value={88}
                        viz={
                            <Chart
                                label="CPU"
                                tone="warn"
                                values={[40, 52, 61, 70, 77, 80, 84, 88]}
                            />
                        }
                    />
                    <Tile
                        aside={{ text: 'On', tone: 'ok' }}
                        foot="on power"
                        label="Awake, this Mac"
                        unit="h left"
                        value={5}
                    />
                    <Tile
                        foot="of 994"
                        label="Disk free"
                        unit="GB"
                        value={582}
                        viz={<Bar label="Disk" percent={41} width="fixed" />}
                    />
                </Tiles>
            </div>
            <div className={styles.meter}>
                <Bar identity="purple" label="Memory of gate" percent={72} />
            </div>
            <div className={styles.meter}>
                <Bar identity="teal" label="Memory of gate2" percent={91} tone="warn" />
            </div>
            <ChartSamples />
        </Section>
    );
}

function ChartSamples() {
    return (
        <>
            <Chart label="CPU of relay" size="row" values={[3, 4, 3, 5, 4, 3, 2, 3]} />
            <Chart
                label="CPU of relay, asleep twice"
                values={[30, 42, null, null, 38, 35, null, 40]}
            />
            <div className={styles.stack}>
                <Chart
                    label="Network"
                    size="activity"
                    tone="new"
                    values={Array.from({ length: 60 }, (_, index) => 20 + ((index * 37) % 60))}
                />
            </div>
        </>
    );
}
