import { AppIcon } from '../../presentation/ui/app-icon';
import { APP_ICON_STORIES } from '../../presentation/ui/app-icon.stories';
import { Bar, ShareBar } from '../../presentation/ui/bar';
import { BAR_STORIES, SHARE_BAR_STORIES } from '../../presentation/ui/bar.stories';
import { Button } from '../../presentation/ui/button';
import { BUTTON_STORIES } from '../../presentation/ui/button.stories';
import { Caption } from '../../presentation/ui/caption';
import { CAPTION_STORIES } from '../../presentation/ui/caption.stories';
import { Chart } from '../../presentation/ui/chart';
import { CHART_STORIES } from '../../presentation/ui/chart.stories';
import { ChoiceCard } from '../../presentation/ui/choice-card';
import { CHOICE_CARD_STORIES } from '../../presentation/ui/choice-card.stories';
import { CodeBlock } from '../../presentation/ui/code-block';
import { CODE_BLOCK_STORIES } from '../../presentation/ui/code-block.stories';
import { ConfirmDialog } from '../../presentation/ui/confirm-dialog';
import { CONFIRM_DIALOG_STORIES } from '../../presentation/ui/confirm-dialog.stories';
import { CopyButton } from '../../presentation/ui/copy-button';
import { COPY_BUTTON_STORIES } from '../../presentation/ui/copy-button.stories';
import { Dialog, Modal } from '../../presentation/ui/dialog';
import { DIALOG_STORIES, MODAL_STORIES } from '../../presentation/ui/dialog.stories';
import { DotStrip } from '../../presentation/ui/dot-strip';
import { DOT_STRIP_STORIES } from '../../presentation/ui/dot-strip.stories';
import { Facts } from '../../presentation/ui/facts';
import { FACTS_STORIES } from '../../presentation/ui/facts.stories';
import { Field } from '../../presentation/ui/field';
import { FIELD_STORIES } from '../../presentation/ui/field.stories';
import { FormField } from '../../presentation/ui/form-field';
import { FORM_FIELD_STORIES } from '../../presentation/ui/form-field.stories';
import { Icon } from '../../presentation/ui/icon';
import { IconButton } from '../../presentation/ui/icon-button';
import { ICON_BUTTON_STORIES } from '../../presentation/ui/icon-button.stories';
import { ICON_STORIES } from '../../presentation/ui/icon.stories';
import { MachineChip } from '../../presentation/ui/machine-chip';
import { MACHINE_CHIP_STORIES } from '../../presentation/ui/machine-chip.stories';
import { MachineIcon } from '../../presentation/ui/machine-icon';
import { MACHINE_ICON_STORIES } from '../../presentation/ui/machine-icon.stories';
import { MemoryGrid } from '../../presentation/ui/memory-grid';
import { MEMORY_GRID_STORIES } from '../../presentation/ui/memory-grid.stories';
import { MemoryPick } from '../../presentation/ui/memory-pick';
import { MEMORY_PICK_STORIES } from '../../presentation/ui/memory-pick.stories';
import { Missing, MissingMark } from '../../presentation/ui/missing';
import { MISSING_MARK_STORIES, MISSING_STORIES } from '../../presentation/ui/missing.stories';
import { NavItem } from '../../presentation/ui/nav-item';
import { NAV_ITEM_STORIES } from '../../presentation/ui/nav-item.stories';
import { Panel } from '../../presentation/ui/panel';
import { PANEL_STORIES } from '../../presentation/ui/panel.stories';
import { Picks } from '../../presentation/ui/picks';
import { PICKS_STORIES } from '../../presentation/ui/picks.stories';
import { ProgressLine } from '../../presentation/ui/progress-line';
import { PROGRESS_LINE_STORIES } from '../../presentation/ui/progress-line.stories';
import { QrCode } from '../../presentation/ui/qr-code';
import { QR_CODE_STORIES } from '../../presentation/ui/qr-code.stories';
import { Row, RowHead } from '../../presentation/ui/row';
import { ROW_HEAD_STORIES, ROW_STORIES } from '../../presentation/ui/row.stories';
import { SearchField } from '../../presentation/ui/search-field';
import { SEARCH_FIELD_STORIES } from '../../presentation/ui/search-field.stories';
import { StateDot } from '../../presentation/ui/state-dot';
import { STATE_DOT_STORIES } from '../../presentation/ui/state-dot.stories';
import { StatePill } from '../../presentation/ui/state-pill';
import { STATE_PILL_STORIES } from '../../presentation/ui/state-pill.stories';
import { Switch } from '../../presentation/ui/switch';
import { SWITCH_STORIES } from '../../presentation/ui/switch.stories';
import { Tile } from '../../presentation/ui/tile';
import { TILE_STORIES } from '../../presentation/ui/tile.stories';
import { TypeTag } from '../../presentation/ui/type-tag';
import { TYPE_TAG_STORIES } from '../../presentation/ui/type-tag.stories';
import { Value } from '../../presentation/ui/value';
import { VALUE_STORIES } from '../../presentation/ui/value.stories';
import { Verdict } from '../../presentation/ui/verdict';
import { VERDICT_STORIES } from '../../presentation/ui/verdict.stories';
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
    entry('MemoryPick', 'Measures', {
        Component: MemoryPick,
        frame: 'block',
        stories: MEMORY_PICK_STORIES,
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
