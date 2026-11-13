import { driftAsked, targetOf } from '../../model/machines';
import type { Found } from '../../model/machines';
import type { Route, Tab } from '../../shell/route';
import { AccountsTab } from '../accounts/accounts';
import { ActivityTab } from '../activity/activity';
import { LogsTab } from '../logs/logs';
import { StorageTab } from '../machine-storage/machine-storage';
import { OverviewTab } from '../overview/overview';
import { PackagesTab } from '../packages/packages';
import { PortsTab } from '../ports/ports';

type Props = { found: Found; tab: Tab; go: (to: Route) => void };

export function MachineTab({ found, tab, go }: Props) {
    const name = found.vm?.Name ?? found.mac.Name;
    switch (tab) {
        case 'Overview': {
            return <OverviewTab found={found} go={go} />;
        }
        case 'Activity': {
            return <ActivityTab found={found} go={go} />;
        }
        case 'Packages': {
            return (
                <PackagesTab
                    asked={driftAsked(found)}
                    key={name}
                    name={name}
                    target={targetOf(found)}
                />
            );
        }
        case 'Ports': {
            return <PortsTab found={found} go={go} />;
        }
        case 'Accounts': {
            return <AccountsTab found={found} go={go} />;
        }
        case 'Storage': {
            return (
                <StorageTab
                    go={go}
                    key={name}
                    name={name}
                    target={targetOf(found)}
                    vm={found.vm !== undefined}
                />
            );
        }
        case 'Logs': {
            return <LogsTab found={found} />;
        }
    }
}
