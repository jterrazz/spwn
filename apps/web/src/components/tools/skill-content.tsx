'use client';

import { inlineRuns, skillBlocks } from './skill-markdown';
import type { SkillBlock } from './skill-markdown';

const ITEM = 'text-muted-foreground/50 my-0.5 flex gap-2 pl-2 text-[12px] leading-relaxed';

function Inline({ text }: { text: string }) {
    return (
        <>
            {inlineRuns(text).map((run, index) =>
                run.code ? (
                    <code
                        className="text-foreground/60 rounded border border-white/[0.08] bg-white/[0.05] px-1 py-0.5 font-mono text-[11px]"
                        // oxlint-disable-next-line react/no-array-index-key -- the runs of one line never move; their place is their identity
                        key={`code-${index}`}
                    >
                        {run.text}
                    </code>
                ) : (
                    run.text
                ),
            )}
        </>
    );
}

function Block({ block }: { block: SkillBlock }) {
    switch (block.kind) {
        case 'code': {
            return (
                <pre className="text-foreground/60 my-3 overflow-x-auto rounded-lg border border-white/[0.06] bg-white/[0.03] px-4 py-3 font-mono text-[11px] leading-relaxed">
                    {block.text}
                </pre>
            );
        }
        case 'heading': {
            return block.level === 2 ? (
                <h3 className="text-foreground/70 mt-6 mb-2 text-sm font-medium">{block.text}</h3>
            ) : (
                <h4 className="text-foreground/60 mt-5 mb-2 text-xs font-medium">{block.text}</h4>
            );
        }
        case 'item': {
            return (
                <div className={ITEM}>
                    <span className="text-muted-foreground/25 w-4 shrink-0 text-right">
                        {block.marker}
                    </span>
                    <span>
                        <Inline text={block.text} />
                    </span>
                </div>
            );
        }
        default: {
            return (
                <p className="text-muted-foreground/50 my-2 text-[12px] leading-relaxed">
                    <Inline text={block.text} />
                </p>
            );
        }
    }
}

/** A skill file's markdown, rendered in the tool page's type scale. */
export function SkillContent({ content }: { content: string }) {
    return (
        <>
            {skillBlocks(content).map((block, index) => (
                // oxlint-disable-next-line react/no-array-index-key -- a skill file renders once, in order; a block's place is its identity
                <Block block={block} key={index} />
            ))}
        </>
    );
}
