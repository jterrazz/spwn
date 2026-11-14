import { AppIcon, Caption, DotStrip, TypeTag, Value } from '../../design/index';
import type { PlanLine } from '../../engine/provisional';
import { engineShown, kindChipWord } from '../../engine/shown';
import { ANY_VERSION_WORD } from '../../engine/words';
import { SEPARATOR } from '../../separator';
import type { Tone } from '../../tone';
import { countWord } from './model';
import type { Layer, Shelf } from './model';

import styles from './layers.module.css';

export type Look =
    | {
          kind: 'file';
          descriptions: ReadonlyMap<string, string>;
          file: string;
          open: (name: string) => void;
      }
    | { kind: 'machine'; differing: ReadonlySet<PlanLine> };

const NO_LAYER_WORD = 'in no layer';
const THIS_FILE_WORD = 'this file';
const EXTENDED_WORD = 'extended';

function keyOf(line: PlanLine): string {
    return `${line.Step.Kind}/${line.Step.Package}`;
}

function Name({ line, kinded }: { line: PlanLine; kinded: boolean }) {
    const { Step } = line;
    const kind = kinded ? kindChipWord(Step.Kind) : '';
    return (
        <span className={styles.name}>
            <AppIcon kind={Step.Kind} name={Step.Package} />
            <span className={styles.title}>{Step.Package}</span>
            {kind === '' ? null : <TypeTag>{kind}</TypeTag>}
        </span>
    );
}

function notOnTheMachine(line: PlanLine, differing: ReadonlySet<PlanLine>): boolean {
    return differing.has(line) && line.Seen.Version === '';
}

function MachineLine({ line, differs }: { line: PlanLine; differs: boolean }) {
    const { Seen } = line;
    const condition = engineShown('MachinefileCondition', Seen.Condition);
    return (
        <div className={`${styles.line} ${styles.machine}`}>
            <Name kinded={false} line={line} />
            <span className={styles.chip}>
                {Seen.Version === '' ? null : <Value look="raised">{Seen.Version}</Value>}
                {Seen.Version === '' && differs ? (
                    <Value look="raised" tone={condition.tone}>
                        {condition.word}
                    </Value>
                ) : null}
            </span>
            <span />
        </div>
    );
}

type FileLineProps = { line: PlanLine; description: string; open: (name: string) => void };

function FileLine({ line, description, open }: FileLineProps) {
    return (
        <button
            className={`${styles.line} ${styles.file} ${styles.opens}`}
            onClick={() => {
                open(line.Step.Package);
            }}
            type="button"
        >
            <Name kinded line={line} />
            <span className={styles.description}>{description}</span>
            <span className={styles.end}>
                <Value look="boxed">
                    {line.Step.Version === '' ? ANY_VERSION_WORD : line.Step.Version}
                </Value>
            </span>
        </button>
    );
}

type LayerHeadProps = { layer: Layer; shelf: Shelf; look: Look };

function hollowOf(layer: Layer, look: Look): ReadonlyMap<string, Tone> {
    if (look.kind !== 'machine') {
        return new Map();
    }
    return new Map(
        layer.lines
            .filter((line) => notOnTheMachine(line, look.differing))
            .map((line) => [
                line.Step.Package,
                engineShown('MachinefileCondition', line.Seen.Condition).tone,
            ]),
    );
}

function countOf(layer: Layer, shelf: Shelf, look: Look): string {
    if (look.kind === 'machine' || layer.name === '') {
        return countWord(layer.lines.length, shelf);
    }
    const relation = layer.name === look.file ? THIS_FILE_WORD : EXTENDED_WORD;
    return `${relation}${SEPARATOR}${String(layer.lines.length)}`;
}

function LayerHead({ layer, shelf, look }: LayerHeadProps) {
    return (
        <Caption>
            <Value identity={layer.identity} look="raised">
                {layer.name === '' ? NO_LAYER_WORD : layer.name}
            </Value>
            <span>{countOf(layer, shelf, look)}</span>
            {look.kind === 'machine' ? (
                <DotStrip
                    hollow={hollowOf(layer, look)}
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
                    <LayerHead layer={layer} look={look} shelf={shelf} />
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
                                open={look.open}
                            />
                        ),
                    )}
                </section>
            ))}
        </div>
    );
}
