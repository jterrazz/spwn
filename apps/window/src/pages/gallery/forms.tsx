import { useState } from 'react';

import { Button, ChoiceCard, Dialog, FormField, Screen, SearchField } from '../../design/index';
import { nothing, Section } from './section';

import styles from './gallery.module.css';

export function Fields() {
    const [query, setQuery] = useState('');
    const [name, setName] = useState('gate5');
    return (
        <Section title="Search field and form field">
            <div className={styles.stack}>
                <SearchField
                    label="Filter"
                    onChange={setQuery}
                    placeholder="Filter the catalog"
                    value={query}
                />
                <FormField
                    hint="Lowercase letters, digits and dashes"
                    label="Name"
                    onChange={setName}
                    value={name}
                />
            </div>
        </Section>
    );
}

export function ChoiceCards() {
    return (
        <Section title="Choice card">
            <div className={styles.cards}>
                <ChoiceCard
                    count="15 apps"
                    description="What a machine runs: tools, runtimes and Mac apps"
                    foot="node, go, docker and 12 more"
                    icon="apps"
                    onChoose={nothing}
                    title="Apps"
                />
                <ChoiceCard
                    count="2 skills"
                    description="What an agent knows how to do"
                    foot="shadcn, web-design"
                    icon="skills"
                    onChoose={nothing}
                    title="Skills"
                />
                <ChoiceCard
                    count="7 settings"
                    description="How a machine behaves"
                    foot="the Dock, the battery and 5 more"
                    icon="settings"
                    onChoose={nothing}
                    title="Settings"
                />
            </div>
        </Section>
    );
}

export function DialogSample() {
    const [dialog, setDialog] = useState(false);
    return (
        <Section title="Dialog">
            <Button
                onClick={() => {
                    setDialog(true);
                }}
            >
                Open the dialog
            </Button>
            {dialog ? (
                <Screen>
                    <Dialog
                        actions={
                            <>
                                <Button
                                    onClick={() => {
                                        setDialog(false);
                                    }}
                                >
                                    Keep gate3
                                </Button>
                                <Button variant="primary">Delete gate3</Button>
                            </>
                        }
                        onClose={() => {
                            setDialog(false);
                        }}
                        title="Delete gate3?"
                    >
                        <FormField
                            label="Type its name to confirm"
                            onChange={nothing}
                            placeholder="gate3"
                            value=""
                        />
                    </Dialog>
                </Screen>
            ) : null}
        </Section>
    );
}
