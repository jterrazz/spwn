type Container = {
    exec: (command: string) => Promise<{ stdout: { text: string } }>;
    inspect: { value: unknown };
    running: boolean;
};

function fieldOf(value: unknown, key: string): unknown {
    if (typeof value !== 'object' || value === null) {
        return undefined;
    }
    const field: unknown = Reflect.get(value, key);
    return field;
}

/** The labels Docker reports for a container, each one a string. */
export function labelsOf(inspected: unknown): Record<string, string> {
    const labels = fieldOf(fieldOf(inspected, 'Config'), 'Labels');
    return typeof labels === 'object' && labels !== null
        ? Object.fromEntries(
              Object.entries(labels).filter(
                  (entry): entry is [string, string] => typeof entry[1] === 'string',
              ),
          )
        : {};
}

/** What a running world is made of, read from Docker and from inside the container, for a spec to compare whole. */
export async function worldOf(container: Container) {
    const inspected = container.inspect.value;
    const labels = labelsOf(inspected);
    const hostConfig = fieldOf(inspected, 'HostConfig');
    const mounts = fieldOf(inspected, 'Mounts');
    const inside = await container.exec(
        "sh -c 'id -un && ls -A /agents/neo && ls -A /world && command -v bash && command -v git'",
    );
    return {
        inside: inside.stdout.text,
        labels: {
            config: labels['sh.spwn.world.config'],
            id: labels['sh.spwn.world.id'],
            kind: labels['sh.spwn.kind'],
            runtime: labels['sh.spwn.world.runtime'],
        },
        mounts: (Array.isArray(mounts) ? mounts : [])
            .map((mount: unknown) => {
                const destination = String(fieldOf(mount, 'Destination')).replace(
                    /^\/workspaces\/.+$/u,
                    '/workspaces/<project>',
                );
                return `${destination} ${fieldOf(mount, 'RW') === true ? 'rw' : 'ro'}`;
            })
            .toSorted(),
        network: fieldOf(hostConfig, 'NetworkMode'),
        pids: fieldOf(hostConfig, 'PidsLimit'),
        running: container.running,
    };
}
