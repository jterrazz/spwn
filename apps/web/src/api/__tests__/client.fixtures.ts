/** A profile the agent route answers with every field set, read back unchanged. */
export const NEO = {
    engine: 'claude',
    journal: [{ date: '2026-10-01', summary: 'shipped' }],
    name: 'neo',
    profile: 'pro',
    provider: 'anthropic',
    purpose: 'build',
    role: 'chief',
    skills: ['git'],
    team: 'ops',
    traits: ['calm'],
};

/** The profile of an agent the API answered with its name and a null journal alone: every default. */
export const TRINITY = {
    engine: '',
    journal: [],
    name: 'trinity',
    profile: '',
    provider: '',
    purpose: '',
    role: 'worker',
    skills: [],
    traits: [],
};
