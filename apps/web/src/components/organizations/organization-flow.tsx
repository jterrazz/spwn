'use client';

import { ReactFlow } from '@xyflow/react';
import { useMemo } from 'react';

import '@xyflow/react/dist/style.css';
import type { Organization } from '../../domain/model';
import { flowHeight, flowOf } from './flow';
import { RoleNode } from './role-node';

const NODE_TYPES = { roleNode: RoleNode };
const FIT = { padding: 0.3 };
const PRO = { hideAttribution: true };
const TRANSPARENT = { background: 'transparent' };

/** An organization's chart, fixed in place: no drag, zoom or selection. */
export function OrganizationFlow({ organization }: { organization: Organization }) {
    const { nodes, edges } = useMemo(() => flowOf(organization.roles), [organization.roles]);
    return (
        <div className="overflow-hidden" style={{ height: flowHeight(organization.roles) }}>
            <ReactFlow
                edges={edges}
                elementsSelectable={false}
                fitView
                fitViewOptions={FIT}
                maxZoom={1.5}
                minZoom={0.5}
                nodes={nodes}
                nodesConnectable={false}
                nodesDraggable={false}
                nodeTypes={NODE_TYPES}
                panOnDrag={false}
                proOptions={PRO}
                style={TRANSPARENT}
                zoomOnDoubleClick={false}
                zoomOnPinch={false}
                zoomOnScroll={false}
            />
        </div>
    );
}
