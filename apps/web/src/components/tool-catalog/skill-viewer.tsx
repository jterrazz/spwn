'use client';

import { IconBookFilled } from '@tabler/icons-react';
import { useState } from 'react';

import type { SkillFile } from '../../domain/tools-catalog';
import { SkillContent } from './skill-content';

/** A tool's skill files, one tab each when there are several. */
export function SkillViewer({ skills }: { skills: SkillFile[] }) {
    const [active, setActive] = useState(0);

    const activeSkill = skills[active] ?? skills[0];
    if (activeSkill === undefined) {
        return null;
    }

    return (
        <div className="overflow-hidden rounded-xl border border-white/[0.07]">
            {skills.length > 1 && (
                <div className="flex border-b border-white/[0.06] bg-white/[0.02]">
                    {skills.map((s, i) => (
                        <button
                            className={`-mb-[1px] flex items-center gap-1.5 border-b-2 px-4 py-2.5 font-mono text-[11px] transition-colors ${
                                active === i
                                    ? 'text-foreground/70 border-purple-400/60 bg-purple-500/5'
                                    : 'text-muted-foreground/30 hover:text-muted-foreground/50 border-transparent'
                            }`}
                            key={s.name}
                            onClick={() => {
                                setActive(i);
                            }}
                            type="button"
                        >
                            <IconBookFilled size={10} />
                            {s.name}
                        </button>
                    ))}
                </div>
            )}

            {skills.length === 1 && (
                <div className="flex items-center gap-2 border-b border-white/[0.06] bg-white/[0.02] px-5 py-3">
                    <IconBookFilled className="text-purple-400/50" size={12} />
                    <span className="text-muted-foreground/40 font-mono text-[11px]">
                        {activeSkill.name}
                    </span>
                </div>
            )}

            <div className="px-5 py-4">
                <SkillContent content={activeSkill.content} />
            </div>
        </div>
    );
}
