import { OPEN_BUTTON_STORIES } from '../../kit/open-button.stories';
import { AppIcon } from '../../ui/app-icon';
import { APP_ICON_STORIES } from '../../ui/app-icon.stories';
import { Bar, ShareBar } from '../../ui/bar';
import { BAR_STORIES, SHARE_BAR_STORIES } from '../../ui/bar.stories';
import { Button } from '../../ui/button';
import { BUTTON_STORIES } from '../../ui/button.stories';
import { Caption } from '../../ui/caption';
import { CAPTION_STORIES } from '../../ui/caption.stories';
import { Chart } from '../../ui/chart';
import { CHART_STORIES } from '../../ui/chart.stories';
import { ChoiceCard } from '../../ui/choice-card';
import { CHOICE_CARD_STORIES } from '../../ui/choice-card.stories';
import { CodeBlock } from '../../ui/code-block';
import { CODE_BLOCK_STORIES } from '../../ui/code-block.stories';
import { ConfirmDialog } from '../../ui/confirm-dialog';
import { CONFIRM_DIALOG_STORIES } from '../../ui/confirm-dialog.stories';
import { CopyButton } from '../../ui/copy-button';
import { COPY_BUTTON_STORIES } from '../../ui/copy-button.stories';
import { Dialog, Modal } from '../../ui/dialog';
import { DIALOG_STORIES, MODAL_STORIES } from '../../ui/dialog.stories';
import { DotStrip } from '../../ui/dot-strip';
import { DOT_STRIP_STORIES } from '../../ui/dot-strip.stories';
import { Facts } from '../../ui/facts';
import { FACTS_STORIES } from '../../ui/facts.stories';
import { Field } from '../../ui/field';
import { FIELD_STORIES } from '../../ui/field.stories';
import { FleetBoard } from '../../ui/fleet-board';
import { FLEET_BOARD_STORIES } from '../../ui/fleet-board.stories';
import { FormField } from '../../ui/form-field';
import { FORM_FIELD_STORIES } from '../../ui/form-field.stories';
import { Icon } from '../../ui/icon';
import { IconButton } from '../../ui/icon-button';
import { ICON_BUTTON_STORIES } from '../../ui/icon-button.stories';
import { ICON_STORIES } from '../../ui/icon.stories';
import { MachineChip } from '../../ui/machine-chip';
import { MACHINE_CHIP_STORIES } from '../../ui/machine-chip.stories';
import { MachineIcon } from '../../ui/machine-icon';
import { MACHINE_ICON_STORIES } from '../../ui/machine-icon.stories';
import { MemoryGrid } from '../../ui/memory-grid';
import { MEMORY_GRID_STORIES } from '../../ui/memory-grid.stories';
import { MemoryPick } from '../../ui/memory-pick';
import { MEMORY_PICK_STORIES } from '../../ui/memory-pick.stories';
import { MemoryTrack } from '../../ui/memory-track';
import { MEMORY_TRACK_STORIES } from '../../ui/memory-track.stories';
import { Missing, MissingMark } from '../../ui/missing';
import { MISSING_MARK_STORIES, MISSING_STORIES } from '../../ui/missing.stories';
import { NavItem } from '../../ui/nav-item';
import { NAV_ITEM_STORIES } from '../../ui/nav-item.stories';
import { Panel } from '../../ui/panel';
import { PANEL_STORIES } from '../../ui/panel.stories';
import { Picks } from '../../ui/picks';
import { PICKS_STORIES } from '../../ui/picks.stories';
import { ProgressLine } from '../../ui/progress-line';
import { PROGRESS_LINE_STORIES } from '../../ui/progress-line.stories';
import { QrCode } from '../../ui/qr-code';
import { QR_CODE_STORIES } from '../../ui/qr-code.stories';
import { RadioRows } from '../../ui/radio-rows';
import { RADIO_ROWS_STORIES } from '../../ui/radio-rows.stories';
import { Row, RowHead } from '../../ui/row';
import { ROW_HEAD_STORIES, ROW_STORIES } from '../../ui/row.stories';
import { SearchField } from '../../ui/search-field';
import { SEARCH_FIELD_STORIES } from '../../ui/search-field.stories';
import { StateDot } from '../../ui/state-dot';
import { STATE_DOT_STORIES } from '../../ui/state-dot.stories';
import { StatePill } from '../../ui/state-pill';
import { STATE_PILL_STORIES } from '../../ui/state-pill.stories';
import { StateSign } from '../../ui/state-sign';
import { STATE_SIGN_STORIES } from '../../ui/state-sign.stories';
import { Switch } from '../../ui/switch';
import { SWITCH_STORIES } from '../../ui/switch.stories';
import { Tile } from '../../ui/tile';
import { TILE_STORIES } from '../../ui/tile.stories';
import { TypeTag } from '../../ui/type-tag';
import { TYPE_TAG_STORIES } from '../../ui/type-tag.stories';
import { Value } from '../../ui/value';
import { VALUE_STORIES } from '../../ui/value.stories';
import { Verdict } from '../../ui/verdict';
import { VERDICT_STORIES } from '../../ui/verdict.stories';
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
    entry('StateSign', 'Marks', { Component: StateSign, stories: STATE_SIGN_STORIES }),
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
    entry('MemoryPick', 'Measures', {
        Component: MemoryPick,
        frame: 'block',
        stories: MEMORY_PICK_STORIES,
    }),
    entry('MemoryTrack', 'Measures', {
        Component: MemoryTrack,
        frame: 'meter',
        stories: MEMORY_TRACK_STORIES,
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
    entry('Picks', 'Controls', { Component: Picks, frame: 'block', stories: PICKS_STORIES }),
    entry('RadioRows', 'Controls', {
        Component: RadioRows,
        frame: 'block',
        stories: RADIO_ROWS_STORIES,
    }),
    entry('NavItem', 'Controls', { Component: NavItem, frame: 'nav', stories: NAV_ITEM_STORIES }),
    entry('SearchField', 'Controls', {
        Component: SearchField,
        frame: 'block',
        stories: SEARCH_FIELD_STORIES,
    }),
    entry('Field', 'Controls', { Component: Field, frame: 'block', stories: FIELD_STORIES }),
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
    entry('FleetBoard', 'Containers', {
        Component: FleetBoard,
        frame: 'block',
        stories: FLEET_BOARD_STORIES,
    }),
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
