import { AppIcon, Caption, DotStrip, StatePill, TypeTag, Value } from '../../design/index';
import type { PlanLine } from '../../engine/provisional';
import { CONDITION_TONES } from '../../engine/tones';
import { ANY_VERSION_WORD, CATALOG_KIND_WORDS, CONDITION_WORDS } from '../../engine/words';
import { countWord } from './model';
import type { Layer, Shelf } from './model';

import styles from './layers.module.css';

export type Look =
    | { kind: 'file'; descriptions: ReadonlyMap<string, string> }
    | { kind: 'machine'; differing: ReadonlySet<PlanLine> };

const NO_LAYER_WORD = 'in no layer';

function keyOf(line: PlanLine): string {
    return `${line.Step.Kind}/${line.Step.Package}`;
}

function Name({ line, kinded }: { line: PlanLine; kinded: boolean }) {
    const { Step } = line;
    const kind = !kinded || Step.Kind === '' ? '' : CATALOG_KIND_WORDS[Step.Kind];
    return (
        <span className={styles.name}>
            <AppIcon kind={Step.Kind} name={Step.Package} />
            <span className={styles.title}>{Step.Package}</span>
            {kind === '' ? null : <TypeTag>{kind}</TypeTag>}
        </span>
    );
}

function MachineLine({ line, differs }: { line: PlanLine; differs: boolean }) {
    const { Seen } = line;
    return (
        <div className={`${styles.line} ${styles.machine}`}>
            <Name kinded={false} line={line} />
            <span className={styles.chip}>
                {differs ? (
                    <StatePill
                        tone={CONDITION_TONES[Seen.Condition]}
                        word={CONDITION_WORDS[Seen.Condition]}
                    />
                ) : Seen.Version === '' ? null : (
                    <Value look="raised">{Seen.Version}</Value>
                )}
            </span>
            <span />
        </div>
    );
}

function FileLine({ line, description }: { line: PlanLine; description: string }) {
    return (
        <div className={`${styles.line} ${styles.file}`}>
            <Name kinded line={line} />
            <span className={styles.description}>{description}</span>
            <span className={styles.end}>
                <Value look="boxed">
                    {line.Step.Version === '' ? ANY_VERSION_WORD : line.Step.Version}
                </Value>
            </span>
        </div>
    );
}

function LayerHead({ layer, shelf, dots }: { layer: Layer; shelf: Shelf; dots: boolean }) {
    return (
        <Caption>
            <Value identity={layer.identity} look="raised">
                {layer.name === '' ? NO_LAYER_WORD : layer.name}
            </Value>
            <span>{countWord(layer.lines.length, shelf)}</span>
            {dots ? (
                <DotStrip
                    identity={layer.identity}
                    names={layer.lines.map((line) => line.Step.Package)}
                />
            ) : null}
        </Caption>
    );
}

export function LayerList({
    layers,
    shelf,
    look,
}: {
    layers: readonly Layer[];
    shelf: Shelf;
    look: Look;
}) {
    if (layers.length === 0) {
        return <p className={styles.none}>None.</p>;
    }
    return (
        <div className={styles.layers}>
            {layers.map((layer) => (
                <section
                    aria-label={layer.name === '' ? NO_LAYER_WORD : layer.name}
                    className={styles.layer}
                    key={layer.name}
                >
                    <LayerHead dots={look.kind === 'machine'} layer={layer} shelf={shelf} />
                    {layer.lines.map((line) =>
                        look.kind === 'machine' ? (
                            <MachineLine
                                differs={look.differing.has(line)}
                                key={keyOf(line)}
                                line={line}
                            />
                        ) : (
                            <FileLine
                                description={look.descriptions.get(line.Step.Package) ?? ''}
                                key={keyOf(line)}
                                line={line}
                            />
                        ),
                    )}
                </section>
            ))}
        </div>
    );
}
