import { AppIcon } from '../../design/app-icon';
import { APP_ICON_STORIES } from '../../design/app-icon.stories';
import { Bar, ShareBar } from '../../design/bar';
import { BAR_STORIES, SHARE_BAR_STORIES } from '../../design/bar.stories';
import { Button } from '../../design/button';
import { BUTTON_STORIES } from '../../design/button.stories';
import { Caption } from '../../design/caption';
import { CAPTION_STORIES } from '../../design/caption.stories';
import { Chart } from '../../design/chart';
import { CHART_STORIES } from '../../design/chart.stories';
import { ChoiceCard } from '../../design/choice-card';
import { CHOICE_CARD_STORIES } from '../../design/choice-card.stories';
import { CodeBlock } from '../../design/code-block';
import { CODE_BLOCK_STORIES } from '../../design/code-block.stories';
import { ConfirmDialog } from '../../design/confirm-dialog';
import { CONFIRM_DIALOG_STORIES } from '../../design/confirm-dialog.stories';
import { CopyButton } from '../../design/copy-button';
import { COPY_BUTTON_STORIES } from '../../design/copy-button.stories';
import { Dialog, Modal } from '../../design/dialog';
import { DIALOG_STORIES, MODAL_STORIES } from '../../design/dialog.stories';
import { DotStrip } from '../../design/dot-strip';
import { DOT_STRIP_STORIES } from '../../design/dot-strip.stories';
import { Facts } from '../../design/facts';
import { FACTS_STORIES } from '../../design/facts.stories';
import { FormField } from '../../design/form-field';
import { FORM_FIELD_STORIES } from '../../design/form-field.stories';
import { Icon } from '../../design/icon';
import { IconButton } from '../../design/icon-button';
import { ICON_BUTTON_STORIES } from '../../design/icon-button.stories';
import { ICON_STORIES } from '../../design/icon.stories';
import { MachineChip } from '../../design/machine-chip';
import { MACHINE_CHIP_STORIES } from '../../design/machine-chip.stories';
import { MachineIcon } from '../../design/machine-icon';
import { MACHINE_ICON_STORIES } from '../../design/machine-icon.stories';
import { MemoryGrid } from '../../design/memory-grid';
import { MEMORY_GRID_STORIES } from '../../design/memory-grid.stories';
import { Missing, MissingMark } from '../../design/missing';
import { MISSING_MARK_STORIES, MISSING_STORIES } from '../../design/missing.stories';
import { NavItem } from '../../design/nav-item';
import { NAV_ITEM_STORIES } from '../../design/nav-item.stories';
import { Panel } from '../../design/panel';
import { PANEL_STORIES } from '../../design/panel.stories';
import { ProgressLine } from '../../design/progress-line';
import { PROGRESS_LINE_STORIES } from '../../design/progress-line.stories';
import { QrCode } from '../../design/qr-code';
import { QR_CODE_STORIES } from '../../design/qr-code.stories';
import { Row, RowHead } from '../../design/row';
import { ROW_HEAD_STORIES, ROW_STORIES } from '../../design/row.stories';
import { SearchField } from '../../design/search-field';
import { SEARCH_FIELD_STORIES } from '../../design/search-field.stories';
import { StateDot } from '../../design/state-dot';
import { STATE_DOT_STORIES } from '../../design/state-dot.stories';
import { StatePill } from '../../design/state-pill';
import { STATE_PILL_STORIES } from '../../design/state-pill.stories';
import { Switch } from '../../design/switch';
import { SWITCH_STORIES } from '../../design/switch.stories';
import { Tile } from '../../design/tile';
import { TILE_STORIES } from '../../design/tile.stories';
import { TypeTag } from '../../design/type-tag';
import { TYPE_TAG_STORIES } from '../../design/type-tag.stories';
import { Value } from '../../design/value';
import { VALUE_STORIES } from '../../design/value.stories';
import { Verdict } from '../../design/verdict';
import { VERDICT_STORIES } from '../../design/verdict.stories';
import { OPEN_BUTTON_STORIES } from '../kit/open-button.stories';
import { entry } from './entry';
import type { Entry } from './entry';
import { OpenButtonSample } from './open-button-sample';

export const ENTRIES: readonly Entry[] = [
    entry('Icon', 'Marks', { Component: Icon, stories: ICON_STORIES }),
    entry('AppIcon', 'Marks', { Component: AppIcon, stories: APP_ICON_STORIES }),
    entry('MachineIcon', 'Marks', { Component: MachineIcon, stories: MACHINE_ICON_STORIES }),
    entry('MachineChip', 'Marks', { Component: MachineChip, stories: MACHINE_CHIP_STORIES }),
    entry('StateDot', 'Marks', { Component: StateDot, stories: STATE_DOT_STORIES }),
    entry('StatePill', 'Marks', { Component: StatePill, stories: STATE_PILL_STORIES }),
    entry('TypeTag', 'Marks', { Component: TypeTag, stories: TYPE_TAG_STORIES }),
    entry('Value', 'Marks', { Component: Value, frame: 'meter', stories: VALUE_STORIES }),
    entry('DotStrip', 'Marks', { Component: DotStrip, stories: DOT_STRIP_STORIES }),
    entry('Caption', 'Marks', { Component: Caption, frame: 'block', stories: CAPTION_STORIES }),
    entry('Missing', 'Marks', { Component: Missing, frame: 'block', stories: MISSING_STORIES }),
    entry('MissingMark', 'Marks', { Component: MissingMark, stories: MISSING_MARK_STORIES }),
    entry('Bar', 'Measures', { Component: Bar, frame: 'meter', stories: BAR_STORIES }),
    entry('ShareBar', 'Measures', {
        Component: ShareBar,
        frame: 'meter',
        stories: SHARE_BAR_STORIES,
    }),
    entry('Chart', 'Measures', { Component: Chart, frame: 'block', stories: CHART_STORIES }),
    entry('Tile', 'Measures', { Component: Tile, frame: 'meter', stories: TILE_STORIES }),
    entry('MemoryGrid', 'Measures', {
        Component: MemoryGrid,
        frame: 'block',
        stories: MEMORY_GRID_STORIES,
    }),
    entry('ProgressLine', 'Measures', {
        Component: ProgressLine,
        frame: 'block',
        stories: PROGRESS_LINE_STORIES,
    }),
    entry('Button', 'Controls', { Component: Button, stories: BUTTON_STORIES }),
    entry('IconButton', 'Controls', { Component: IconButton, stories: ICON_BUTTON_STORIES }),
    entry('CopyButton', 'Controls', { Component: CopyButton, stories: COPY_BUTTON_STORIES }),
    entry('OpenButton', 'Controls', { Component: OpenButtonSample, stories: OPEN_BUTTON_STORIES }),
    entry('Switch', 'Controls', { Component: Switch, frame: 'block', stories: SWITCH_STORIES }),
    entry('NavItem', 'Controls', { Component: NavItem, frame: 'nav', stories: NAV_ITEM_STORIES }),
    entry('SearchField', 'Controls', {
        Component: SearchField,
        frame: 'block',
        stories: SEARCH_FIELD_STORIES,
    }),
    entry('FormField', 'Controls', {
        Component: FormField,
        frame: 'block',
        stories: FORM_FIELD_STORIES,
    }),
    entry('ChoiceCard', 'Controls', {
        Component: ChoiceCard,
        frame: 'meter',
        stories: CHOICE_CARD_STORIES,
    }),
    entry('Panel', 'Containers', { Component: Panel, frame: 'block', stories: PANEL_STORIES }),
    entry('Row', 'Containers', { Component: Row, frame: 'block', stories: ROW_STORIES }),
    entry('RowHead', 'Containers', {
        Component: RowHead,
        frame: 'block',
        stories: ROW_HEAD_STORIES,
    }),
    entry('Facts', 'Containers', { Component: Facts, frame: 'block', stories: FACTS_STORIES }),
    entry('CodeBlock', 'Containers', {
        Component: CodeBlock,
        frame: 'block',
        stories: CODE_BLOCK_STORIES,
    }),
    entry('QrCode', 'Containers', { Component: QrCode, stories: QR_CODE_STORIES }),
    entry('Verdict', 'Statements', {
        Component: Verdict,
        frame: 'block',
        stories: VERDICT_STORIES,
    }),
    entry('Modal', 'Dialogs', { Component: Modal, frame: 'dialog', stories: MODAL_STORIES }),
    entry('Dialog', 'Dialogs', { Component: Dialog, frame: 'dialog', stories: DIALOG_STORIES }),
    entry('ConfirmDialog', 'Dialogs', {
        Component: ConfirmDialog,
        frame: 'dialog',
        stories: CONFIRM_DIALOG_STORIES,
    }),
];
