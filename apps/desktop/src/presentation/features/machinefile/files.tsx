import { sentence } from '../../../domain/format';
import { SEPARATOR } from '../../../domain/separator';
import type { Tone } from '../../../domain/tone';
import { AppIcon, Caption, DotStrip, TypeTag, Value } from '../../ui/index';
import type { PlanLine } from '../../words/provisional';
import { engineShown, kindChipWord, stepBoxed } from '../../words/shown';
import { countWord } from './model';
import type { PlanFile, Shelf } from './model';

import styles from './files.module.css';

export type Look =
    | {
          kind: 'file';
          descriptions: ReadonlyMap<string, string>;
          file: string;
          open: (name: string) => void;
      }
    | { kind: 'machine'; differing: ReadonlySet<PlanLine> };

const NO_FILE_WORD = 'in no layer';
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
            <span className={styles.description}>{sentence([line.Step.Rule, line.Waiting])}</span>
        </div>
    );
}

type FileLineProps = { line: PlanLine; description: string; open: (name: string) => void };

function FileLine({ line, description, open }: FileLineProps) {
    const boxed = stepBoxed(line.Step);
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
                {boxed === '' ? null : <Value look="boxed">{boxed}</Value>}
            </span>
        </button>
    );
}

type PlanFileHeadProps = { planFile: PlanFile; shelf: Shelf; look: Look };

function hollowOf(planFile: PlanFile, look: Look): ReadonlyMap<string, Tone> {
    if (look.kind !== 'machine') {
        return new Map();
    }
    return new Map(
        planFile.lines
            .filter((line) => notOnTheMachine(line, look.differing))
            .map((line) => [
                line.Step.Package,
                engineShown('MachinefileCondition', line.Seen.Condition).tone,
            ]),
    );
}

function countOf(planFile: PlanFile, shelf: Shelf, look: Look): string {
    if (look.kind === 'machine' || planFile.name === '') {
        return countWord(planFile.lines.length, shelf);
    }
    const relation = planFile.name === look.file ? THIS_FILE_WORD : EXTENDED_WORD;
    return `${relation}${SEPARATOR}${String(planFile.lines.length)}`;
}

function PlanFileHead({ planFile, shelf, look }: PlanFileHeadProps) {
    return (
        <Caption>
            <Value identity={planFile.identity} look="raised">
                {planFile.name === '' ? NO_FILE_WORD : planFile.name}
            </Value>
            <span>{countOf(planFile, shelf, look)}</span>
            {look.kind === 'machine' ? (
                <DotStrip
                    hollow={hollowOf(planFile, look)}
                    identity={planFile.identity}
                    names={planFile.lines.map((line) => line.Step.Package)}
                />
            ) : null}
        </Caption>
    );
}

export function PlanFileList({
    planFiles,
    shelf,
    look,
}: {
    planFiles: readonly PlanFile[];
    shelf: Shelf;
    look: Look;
}) {
    if (planFiles.length === 0) {
        return <p className={styles.none}>None.</p>;
    }
    return (
        <div className={styles.planFiles}>
            {planFiles.map((planFile) => (
                <section
                    aria-label={planFile.name === '' ? NO_FILE_WORD : planFile.name}
                    className={styles.planFile}
                    key={planFile.name}
                >
                    <PlanFileHead look={look} planFile={planFile} shelf={shelf} />
                    {planFile.lines.map((line) =>
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
