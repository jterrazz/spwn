import { Caption, Verdict } from '../../design/index';
import { Page } from '../page';
import { ENTRIES } from './entries';
import { anchorOf, LAYERS } from './entry';
import type { Entry, Layer } from './entry';

import styles from './gallery.module.css';

function anchorOfLayer(layer: Layer): string {
    return `layer-${layer.toLowerCase()}`;
}

function Contents() {
    return (
        <nav aria-label="Contents" className={styles.contents}>
            {LAYERS.map((layer) => (
                <p className={styles.line} key={layer}>
                    <a className={styles.layerLink} href={`#${anchorOfLayer(layer)}`}>
                        {layer}
                    </a>
                    {ENTRIES.filter((one) => one.layer === layer).map((one) => (
                        <a className={styles.link} href={`#${anchorOf(one.name)}`} key={one.name}>
                            {one.name}
                        </a>
                    ))}
                </p>
            ))}
        </nav>
    );
}

function EntryView({ entry }: { entry: Entry }) {
    return (
        <section
            aria-label={entry.name}
            className={styles.entry}
            data-entry={entry.name}
            id={anchorOf(entry.name)}
        >
            <h3 className={styles.name}>{entry.name}</h3>
            <div className={`${styles.samples} ${styles[entry.frame]}`}>
                {entry.samples.map((sample) => (
                    <figure className={styles.sample} key={sample.name}>
                        <div className={styles.drawn}>{sample.drawn}</div>
                        <figcaption className={styles.story}>{sample.name}</figcaption>
                    </figure>
                ))}
            </div>
        </section>
    );
}

export default function Gallery() {
    return (
        <Page>
            <Verdict
                line={`${String(ENTRIES.length)} components, each drawn in its variants and its data states`}
                size="page"
                title="Every component, every variant."
                tone="ok"
                word="Up"
            />
            <Contents />
            {LAYERS.map((layer) => (
                <section
                    aria-label={layer}
                    className={styles.layer}
                    id={anchorOfLayer(layer)}
                    key={layer}
                >
                    <Caption>{layer}</Caption>
                    {ENTRIES.filter((one) => one.layer === layer).map((one) => (
                        <EntryView entry={one} key={one.name} />
                    ))}
                </section>
            ))}
        </Page>
    );
}
