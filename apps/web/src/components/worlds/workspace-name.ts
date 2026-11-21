const WORKSPACE_NAME = /^[a-z][a-z0-9-]*$/u;

/** The message at a workspace name's field, empty when the name is acceptable. A blank name is acceptable: the dialog names the mount itself. The rule is the engine's, `ValidateWorkspaceName`, applied to the trimmed name the dialog sends. */
export function workspaceNameError(name: string): string {
    const trimmed = name.trim();
    if (trimmed === '' || WORKSPACE_NAME.test(trimmed)) {
        return '';
    }
    return 'Use lowercase letters, digits and hyphens, starting with a letter';
}
