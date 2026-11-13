import { useState } from 'react';

import {
    Button,
    BUTTON_VARIANTS,
    IconButton,
    Screen,
    StatePill,
    Switch,
    TONES,
    TypeTag,
    Value,
} from '../../design/index';
import type { Tone } from '../../tone';
import { nothing, Section } from './section';

const SAMPLE_WORDS: Readonly<Record<Tone, string>> = {
    bad: 'Down',
    new: 'Creating',
    ok: 'Running',
    plain: 'Stopped',
    warn: 'Needs you',
};

export function StatePills() {
    return (
        <Section title="State pill, and the quiet word a calm one is in a list">
            {TONES.map((tone) => (
                <StatePill key={tone} tone={tone} word={SAMPLE_WORDS[tone]} />
            ))}
            {TONES.map((tone) => (
                <StatePill key={`quiet/${tone}`} quiet tone={tone} word={SAMPLE_WORDS[tone]} />
            ))}
        </Section>
    );
}

export function Tags() {
    return (
        <Section title="Type tag, value, field">
            <TypeTag>manager</TypeTag>
            <TypeTag>setting</TypeTag>
            <TypeTag>optional</TypeTag>
            <Value>24.11.1</Value>
            <Value>:3000</Value>
            <Value>~/.claude/settings.json</Value>
            <Value look="raised">24.11.1</Value>
            <Value look="boxed">vault:claude</Value>
        </Section>
    );
}

export function Buttons() {
    return (
        <Section title="Button">
            <Screen>
                <Button variant="primary">Open in Terminal</Button>
            </Screen>
            {BUTTON_VARIANTS.filter((variant) => variant !== 'primary').map((variant) => (
                <Button key={variant} variant={variant}>
                    {variant}
                </Button>
            ))}
            <Button disabled>Disabled</Button>
            <Screen>
                <Button disabled variant="primary">
                    Free what can be rebuilt
                </Button>
            </Screen>
        </Section>
    );
}

export function IconButtons() {
    return (
        <Section title="Icon button">
            <IconButton icon="sidebar" label="Show or hide the sidebar" onClick={nothing} />
            <IconButton icon="back" label="Back" onClick={nothing} shortcut="⌘[" />
            <IconButton icon="find" label="Find" onClick={nothing} shortcut="⌘K" />
            <IconButton disabled icon="back" label="Back" onClick={nothing} />
        </Section>
    );
}

const WRAPPED = [
    'Package managers',
    'Runtimes',
    'Shell & terminal',
    'Command line',
    'Git',
    'Editors & IDEs',
    'Containers & VMs',
    'Deploy',
    'AI agents',
    'AI tooling',
    'Browsers',
    'Communication',
    'Productivity',
].map((label) => ({ id: label.toLowerCase(), label }));

export function Switches() {
    const [view, setView] = useState<'built-in' | 'mine' | null>(null);
    const [tab, setTab] = useState<'Activity' | 'Overview' | 'Ports'>('Overview');
    return (
        <Section title="Switch">
            <Switch<'built-in' | 'mine'>
                all="All"
                label="Which files"
                onChange={setView}
                options={[
                    { id: 'mine', label: 'Yours', count: 2 },
                    { id: 'built-in', label: 'Built-in', count: 5 },
                ]}
                value={view}
            />
            <Switch
                label="Tabs"
                onChange={setTab}
                options={[
                    { id: 'Overview', label: 'Overview' },
                    { id: 'Activity', label: 'Activity' },
                    { id: 'Ports', label: 'Ports' },
                ]}
                value={tab}
                variant="tabs"
            />
            <Switch
                disabled
                label="A switch with nothing to choose yet"
                onChange={nothing}
                options={[
                    { id: 'apps', label: 'Apps' },
                    { id: 'settings', label: 'Settings' },
                ]}
                value="apps"
            />
            <Switch
                label="A switch that wraps its options"
                onChange={nothing}
                options={WRAPPED}
                value="runtimes"
                wrap
            />
        </Section>
    );
}
