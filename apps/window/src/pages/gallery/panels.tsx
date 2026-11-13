import {
    Bar,
    Button,
    Caption,
    Chart,
    CodeBlock,
    Facts,
    MachineChip,
    MachineIcon,
    Missing,
    Panel,
    Row,
    RowHead,
    TypeTag,
    Value,
} from '../../design/index';
import { nothing, Section } from './section';

import styles from './gallery.module.css';

function MachinesPanel() {
    return (
        <Panel note="a row with a column template" title="Machines">
            <RowHead
                columns={[
                    { label: 'Machine' },
                    { label: 'CPU', numeric: true },
                    { label: 'Memory, GB', numeric: true },
                ]}
                lead
                template="meters"
            />
            <Row
                cells={[
                    <Chart key="c" label="CPU" size="row" values={[50, 60, 64]} />,
                    <Bar identity="purple" key="b" label="Memory" percent={72} />,
                ]}
                lead={<MachineIcon identity="purple" kind="workspace" />}
                onOpen={nothing}
                template="meters"
                title="gate"
            />
        </Panel>
    );
}

function ColumnsPanel() {
    return (
        <Panel note="a column per part, kept empty where a row has none" title="Columns">
            <Row
                kind="manager"
                sentence="One tool that installs every language runtime"
                state={{ tone: 'ok', word: 'Present' }}
                title="mise"
                value={<Value>2025.11.3</Value>}
            />
            <Row
                kind={null}
                sentence="A calm state is a quiet word; a deviation is a pill"
                state={{ tone: 'warn', word: 'Outdated' }}
                title="node"
                value={<Value>24.11.0</Value>}
            />
            <Row
                kind="cache"
                sentence="A size is plain text with tabular figures"
                state={null}
                title="Docker images"
                value="12.4 GB"
            />
            <Row
                kind={null}
                sentence="A path keeps its end: it is cut in the middle"
                state={{ tone: 'plain', word: 'Not installed' }}
                title="station"
                value={<Value>~/.config/station/machinefiles/jterrazz-macbook-pro.yaml</Value>}
            />
            <Row
                lead={<MachineIcon identity="pink" kind="workspace" />}
                onOpen={nothing}
                sentence="Opens the machine: the whole row is the gesture"
                title="gate4"
            />
            <Row
                link={<MachineChip kind="workspace" name="gate4" onOpen={nothing} />}
                onOpen={nothing}
                sentence="Opens the account; its chip opens the machine that holds it"
                title="GitHub"
            />
        </Panel>
    );
}

export function PanelsAndRows() {
    return (
        <Section title="Panel and row">
            <div className={styles.stack}>
                <Panel
                    action={<Button variant="link">3 more can wait</Button>}
                    note="2"
                    title="Needs you"
                >
                    <Row
                        sentence="jterrazz-macbook-pro · AirPlay Receiver · not declared public"
                        state={{ tone: 'bad', word: 'Exposed' }}
                        stateAt="start"
                        title="Port 5000 is reachable from your local network"
                        trail={<Button>Review</Button>}
                    />
                    <Row
                        sentence="workspace · shared 11 h ago"
                        state={{ tone: 'warn', word: 'Needs you' }}
                        stateAt="start"
                        title="GitHub shared with workspace ends in 40 min"
                        trail={<Button>Extend 12 h</Button>}
                    />
                    <Row
                        size="slim"
                        title="node 24.11.0 → 24.11.1"
                        trail={<TypeTag>6 machines</TypeTag>}
                    />
                </Panel>
                <ColumnsPanel />
                <MachinesPanel />
                <Panel padded title="A captioned panel">
                    <Caption>A caption heads a group of rows</Caption>
                </Panel>
                <Panel note="it says so" title="An empty panel" />
                <Panel title="A field no call gives">
                    <Missing>What no call answers is named here, as the field it lacks.</Missing>
                </Panel>
            </div>
        </Section>
    );
}

export function FactsSample() {
    return (
        <Section title="Facts">
            <div className={styles.stack}>
                <Facts
                    facts={[
                        { label: 'Purpose', value: 'Design of the Station window' },
                        { label: 'Created', value: '6 Nov, by studio-os-e8' },
                        { label: 'Hold', value: 'Yours' },
                        { label: 'Image', value: <Value>cirruslabs/ubuntu:24.04</Value> },
                    ]}
                />
                <Facts
                    facts={[
                        { label: 'System', value: 'Ubuntu 24.04 · arm64' },
                        { label: 'Resources', value: '4 CPU · 4 GB · 20 GB disk' },
                    ]}
                    layout="column"
                />
                <Facts
                    facts={[
                        {
                            label: 'Name on the tailnet',
                            value: 'a-long-machine-name.tail1234.ts.net',
                        },
                        { label: 'Serve', value: 'on' },
                        { label: 'Range', value: '40000 to 40999 · 998 free · 20 per machine' },
                    ]}
                    layout="line"
                />
            </div>
        </Section>
    );
}

export function CodeBlocks() {
    return (
        <Section title="Code block">
            <div className={styles.stack}>
                <CodeBlock
                    label="A log"
                    lines={[
                        { number: 1, text: 'station engine started' },
                        { change: 'removed', number: 2, text: 'relay check failed' },
                        { number: 3, text: 'relay restarted' },
                    ]}
                />
                <CodeBlock
                    label="A diff"
                    lines={[
                        { number: 1, text: 'extends: [base, dev-tools]' },
                        { number: 2, text: 'apps:' },
                        { change: 'added', number: 3, text: '  - lazygit' },
                        { change: 'removed', number: 4, text: '  - jq' },
                    ]}
                    numbered
                />
            </div>
        </Section>
    );
}
