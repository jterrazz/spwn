import { AppIcon, IDENTITIES, MachineChip, MachineIcon, NavItem } from '../../design/index';
import { nothing, Section } from './section';

import styles from './gallery.module.css';

export function NavItems() {
    return (
        <Section title="Nav item">
            <div className={styles.nav}>
                <NavItem current icon="home" label="Home" onSelect={nothing} />
                <NavItem
                    count={{ value: 2, urgent: true }}
                    current={false}
                    icon="inbox"
                    label="Inbox"
                    onSelect={nothing}
                />
                <NavItem
                    current={false}
                    label="jterrazz-mac-mini"
                    lead={<MachineIcon kind="mac" />}
                    onSelect={nothing}
                    state={{ tone: 'ok', word: 'Up' }}
                />
                <NavItem
                    current={false}
                    indent
                    label="relay"
                    lead={<MachineIcon identity="lilac" kind="server" />}
                    onSelect={nothing}
                    state={{ tone: 'bad', word: 'Down' }}
                />
                <NavItem
                    current={false}
                    indent
                    label="gate3"
                    lead={<MachineIcon identity="indigo" kind="workspace" />}
                    onSelect={nothing}
                    state={{ tone: 'plain', word: 'Stopped' }}
                />
                <NavItem
                    count={{ value: 15, urgent: false }}
                    current={false}
                    icon="apps"
                    indent
                    label="Apps"
                    onSelect={nothing}
                />
            </div>
        </Section>
    );
}

export function Identities() {
    return (
        <Section title="Machine icon and chip, the identity palette">
            {IDENTITIES.map((identity) => (
                <MachineIcon identity={identity} key={identity} kind="workspace" />
            ))}
            <MachineIcon kind="mac" />
            <MachineIcon identity="sky" kind="server" size="large" />
            <MachineChip identity="blue" kind="workspace" name="workspace" onOpen={nothing} />
            <MachineChip kind="mac" name="jterrazz-macbook-pro" onOpen={nothing} />
        </Section>
    );
}

export function AppIcons() {
    return (
        <Section title="App icon: a logo, a monogram when none is held, the glyph of a setting">
            {['github', 'node', 'docker', 'claude', 'uv', 'herdr', 'codex'].map((name) => (
                <AppIcon key={name} name={name} />
            ))}
            <AppIcon kind="setting" name="git-shortcuts" />
        </Section>
    );
}
