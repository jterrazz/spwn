import { describe, expect, test } from 'vitest';

import { deduplicateBlocks, parseStreamEvent, withBlocks } from '../blocks';
import type { ActivityBlock } from '../blocks';

const assistant = (content: unknown[]) => ({ message: { content }, type: 'assistant' });
const user = (content: unknown[]) => ({ message: { content }, type: 'user' });

describe('blocks', () => {
    test('an assistant event keeps its thinking, its text and its complete tool calls, and drops the rest', () => {
        // Given - every block shape an assistant event holds, whole and incomplete
        const event = assistant([
            { text: 'weighing it', type: 'thinking' },
            { text: 'hello', type: 'text' },
            { text: '', type: 'text' },
            { id: 't-1', input: { file_path: 'a.ts' }, name: 'Read', type: 'tool_use' },
            { id: 't-2', name: 'Bash', type: 'tool_use' },
            { id: 't-3', input: { pattern: 'x' }, name: 'Grep', text: 'stray', type: 'tool_use' },
            { id: '', name: 'Bash', type: 'tool_use' },
            { id: 't-4', name: '', type: 'tool_use' },
            { id: 't-5', name: 'Edit', type: 'text' },
            { id: 't-6', name: 'Edit', type: 'image' },
            { text: 'not a block', type: 'image' },
            'junk',
        ]);

        // Then - the thinking and the text, the three whole calls, and nothing for the rest
        expect(parseStreamEvent(event)).toStrictEqual([
            { content: 'weighing it', type: 'thinking' },
            { content: 'hello', type: 'text' },
            { id: 't-1', input: { file_path: 'a.ts' }, tool: 'Read', type: 'tool_use' },
            { id: 't-2', input: {}, tool: 'Bash', type: 'tool_use' },
            { id: 't-3', input: { pattern: 'x' }, tool: 'Grep', type: 'tool_use' },
        ]);
    });

    test('an event with nothing to show reads as none', () => {
        // Given - an assistant and a user event whose content holds nothing, and events of other kinds
        const events = [
            assistant([]),
            assistant([{ text: '', type: 'text' }]),
            user([]),
            user([{ tool_use_id: 't-1', type: 'text' }, { type: 'tool_result' }]),
            { subtype: 'success', type: 'result' },
            { item: { text: '' }, type: 'item.completed' },
            { item: {}, type: 'item.completed' },
            { type: 'system' },
        ];

        // Then - none of them yields a block
        expect(events.map((event) => parseStreamEvent(event))).toStrictEqual(
            events.map(() => null),
        );
    });

    test('a user event carries the results of the tool calls, a failed one marked, a structured one as JSON', () => {
        // Given - a text result, a failed one, a structured one, and two that name no call or no result
        const event = user([
            { content: 'ok', tool_use_id: 't-1', type: 'tool_result' },
            { content: 'denied', is_error: true, tool_use_id: 't-2', type: 'tool_result' },
            { content: [{ text: 'a' }], is_error: false, tool_use_id: 't-3', type: 'tool_result' },
            { content: 'orphan', tool_use_id: '', type: 'tool_result' },
            { content: 'wrong type', tool_use_id: 't-4', type: 'text' },
        ]);

        // Then - three results, in order
        expect(parseStreamEvent(event)).toStrictEqual([
            { content: 'ok', id: 't-1', isError: false, type: 'tool_result' },
            { content: 'denied', id: 't-2', isError: true, type: 'tool_result' },
            { content: '[{"text":"a"}]', id: 't-3', isError: false, type: 'tool_result' },
        ]);
    });

    test('a result event shows an error with its words, or an unknown one, and shows nothing otherwise', () => {
        // Given - an error with a message, one without, and a success
        const events = [
            { result: 'rate limited', subtype: 'error', type: 'result' },
            { subtype: 'error', type: 'result' },
            { result: 'done', subtype: 'success', type: 'result' },
        ];

        // Then - the message, the fallback, and none
        expect(events.map((event) => parseStreamEvent(event))).toStrictEqual([
            [{ content: 'rate limited', type: 'error' }],
            [{ content: 'Unknown error', type: 'error' }],
            null,
        ]);
    });

    test('a completed Codex item shows its text', () => {
        // Given - a completed item with text
        const event = { item: { text: 'patched' }, type: 'item.completed' };

        // Then - one text block
        expect(parseStreamEvent(event)).toStrictEqual([{ content: 'patched', type: 'text' }]);
    });

    test('deduplication keeps a tool call by its id, and a text or a thinking while the turn shows none of its kind', () => {
        // Given - a turn that shows two calls, a text and a result, and an event repeating it cumulatively
        const shown: ActivityBlock[] = [
            { id: 't-1', input: {}, tool: 'Read', type: 'tool_use' },
            { id: 't-2', input: {}, tool: 'Bash', type: 'tool_use' },
            { content: 'hello', type: 'text' },
            { content: 'out', id: 't-3', isError: false, type: 'tool_result' },
        ];
        const repeated: ActivityBlock[] = [
            { id: 't-2', input: {}, tool: 'Bash', type: 'tool_use' },
            { id: 't-3', input: {}, tool: 'Grep', type: 'tool_use' },
            { content: 'hello again', type: 'text' },
            { content: 'weighing', type: 'thinking' },
            { content: 'boom', type: 'error' },
            { content: 'out', id: 't-3', isError: false, type: 'tool_result' },
        ];

        // Then - the repeated call and text are dropped; the call whose id only a result holds, the first thinking, the error and the result stay
        expect(deduplicateBlocks(shown, repeated)).toStrictEqual([
            { id: 't-3', input: {}, tool: 'Grep', type: 'tool_use' },
            { content: 'weighing', type: 'thinking' },
            { content: 'boom', type: 'error' },
            { content: 'out', id: 't-3', isError: false, type: 'tool_result' },
        ]);
    });

    test('deduplication drops a thinking once shown, and keeps a text the turn has only thought before', () => {
        // Given - a turn that shows a thinking only, and an event with a thinking and a text
        const shown: ActivityBlock[] = [{ content: 'weighing', type: 'thinking' }];
        const incoming: ActivityBlock[] = [
            { content: 'weighing more', type: 'thinking' },
            { content: 'hello', type: 'text' },
        ];

        // Then - the text is new, the thinking is not
        expect(deduplicateBlocks(shown, incoming)).toStrictEqual([
            { content: 'hello', type: 'text' },
        ]);
    });

    test('a message takes the new blocks after its own, its content the text of them all', () => {
        // Given - a message with a text, and an event adding a call and another text
        const message = {
            blocks: [{ content: 'a', type: 'text' }] satisfies ActivityBlock[],
            content: 'a',
            role: 'agent',
        };
        const added: ActivityBlock[] = [
            { id: 't-1', input: {}, tool: 'Read', type: 'tool_use' },
            { content: 'b', type: 'text' },
        ];

        // Then - the blocks follow in order and the content joins the texts
        expect(withBlocks(message, added)).toStrictEqual({
            blocks: [{ content: 'a', type: 'text' }, ...added],
            content: 'ab',
            role: 'agent',
        });
    });
});
