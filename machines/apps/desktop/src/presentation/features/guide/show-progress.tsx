import { useRead } from '../../dependencies/dependencies';
import { Button } from '../../ui/index';

type Props = { machine: string; onShow: ((act: string) => void) | undefined };

// invariant: a machine whose act is still running offers its Guide again, the one place a hidden Guide is reached
// invariant: until the path bar's chip lists the live acts; a page drawn where no Guide is shown offers nothing.
export function ShowProgress({ machine, onShow }: Props) {
    const guides = useRead('engine.guides', {});
    const live =
        guides.kind === 'read'
            ? (guides.out.guides ?? []).find((one) => one.live && one.subject === machine)
            : undefined;
    if (onShow === undefined || live === undefined) {
        return null;
    }
    return (
        <Button
            onClick={() => {
                onShow(live.id);
            }}
        >
            Show progress
        </Button>
    );
}
