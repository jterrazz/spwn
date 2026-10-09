'use client';

import { IconMessageCircle } from '@tabler/icons-react';

import { useArchitectChat } from '../../contexts/architect-chat-context';
import { Chat } from '../chat';
import { MetricGrid } from '../ds';

/** The architect page while the architect runs: its counts, and the conversation with it. */
export function RunningView() {
    const { architectStatus, chatInput, messages, sending, sendMessage, setChatInput } =
        useArchitectChat();
    const kpis = architectStatus?.kpis;
    return (
        <>
            <MetricGrid
                className="w-fit gap-x-10"
                columns={2}
                items={[
                    { label: 'Worlds', value: kpis?.worlds ?? 0 },
                    { label: 'Agents', value: kpis?.agents ?? 0 },
                ]}
            />
            <Chat
                className="min-h-[320px] flex-1"
                disabled={sending}
                emptyState={
                    <div className="flex flex-col items-center justify-center text-center">
                        <IconMessageCircle className="text-muted-foreground/15 mb-3" size={28} />
                        <p className="text-muted-foreground/30 text-sm">Ask anything</p>
                        <p className="text-muted-foreground/20 mt-1 max-w-sm text-[11px]">
                            &quot;Create an agent for the API project&quot;, &quot;What&apos;s
                            running?&quot;, &quot;Spawn a world for the frontend repo&quot;
                        </p>
                    </div>
                }
                focusInput
                input={chatInput}
                messages={messages.map((message) => ({
                    ...message,
                    role: message.role === 'architect' ? 'assistant' : 'user',
                }))}
                onInputChange={setChatInput}
                onSend={() => {
                    void sendMessage();
                }}
                placeholder="Talk to the Architect..."
                typingText="Architect is thinking..."
            />
        </>
    );
}
