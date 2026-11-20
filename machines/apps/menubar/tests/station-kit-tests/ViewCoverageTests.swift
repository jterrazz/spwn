import XCTest

final class ViewCoverageTests: XCTestCase {
    static let views = ["atom/", "molecule/", "organism/", "template/", "motion/"]
    static let declaration = try! NSRegularExpression(pattern: #"^(?:(?:public|private|fileprivate|internal|final|@MainActor) )*(?:struct|enum|class|extension|func|protocol|actor) "#)
    static let view = try! NSRegularExpression(pattern: #"^(?:(?:public|private|fileprivate|internal) )?struct (\w+)(?:<.*>)?: (?:SwiftUI\.)?View\b"#)

    // value: the kit's views that record nothing in a view golden yet, the
    // recorder's pending output: a view that records leaves it, and the
    // list only shrinks.
    static let pending = [
        "AccountRow", "AccountsPage", "AccountsSection", "ActivityCharts", "ActivityPage", "AddRow", "AlertCard", "AllPorts", "AwakeMac",
        "AwakePage", "BackButton", "BackKey", "Bar", "BirthBar", "Card", "CardLabel", "CareCard", "CellsRow", "CheckBox", "CheckRow", "CheckRows",
        "Chevron", "ChipPicker", "Choice", "CleanupSheet", "CommandBox", "ConfirmPanel", "ConfirmSheet", "CopyFooter", "CopyIcon", "DisclosureRow",
        "EdgeFade", "EmptyState", "ErrorLine", "EscapeKey", "FactView", "FactsLine", "FindEscape", "FindField", "FindFooter", "FindMore",
        "FindPanel", "FindResult", "FixedValue", "FoldButton", "FootNote", "FormRow", "FormSheet", "FormText", "FormView", "GaugeStrip",
        "GaugeTile", "GlyphView", "GrantSheet", "GroupCaption", "GroupView", "HUDCapsule", "HUDFlight", "HealthCard", "HintSheet", "HintView",
        "HomeHead", "HomeHeaderBar", "HomeLead", "HomeMeasure", "HomePage", "HomeSection", "HomeStrip", "HomeVerb", "IconButton", "IconTile",
        "InfoPopover", "ItemView", "ItemsView", "KeyCap", "KindTag", "LatencyCard", "LinkButton", "LinkText", "LinkView", "ListFooter",
        "ListWords", "ListenerRows", "LiveRead", "LiveText", "LogCard", "LogLines", "LogoChip", "MacAccounts", "MacOverview", "MacPage",
        "MacPorts", "MacRing", "MachineActivity", "MachineContent", "MachineHeader", "MachinePage", "MachineRow", "MachineTree", "MarkView",
        "MemorySlider", "MenuGlyph", "MenuLine", "MenuPageFrame", "MenuPreview", "MenuRow", "MenuStage", "MenuText", "Metric", "Metrics",
        "NeedRow", "NeedsCard", "NeedsPage", "NewVMSheet", "OptionGlyph", "PageFooter", "PageHeader", "PanelButton", "PendingDot", "PickRows",
        "Pill", "PlainIcon", "Popover", "PopoverCard", "PopoverLayer", "PopupField", "PortRow", "PortsPage", "PortsSection", "ProgressRing",
        "QRPopover", "QRView", "QuickSlot", "RoleText", "RoundButton", "RowSeparator", "RowSpark", "RowStack", "RowText", "RowView", "Rows",
        "SSHFooter", "SectionDivider", "SectionHeader", "SegmentSection", "Segmented", "ServerPage", "SharePortSheet", "SharedPortCard",
        "SheetButton", "SheetCard", "SheetFor", "SheetLine", "SheetPrimary", "SheetRows", "SkeletonGauges", "Spark", "Spinner", "SplitButton",
        "StateDot", "StateWord", "StationButton", "StationSwitch", "StatusBar", "StatusDot", "StatusItem", "SubText", "Switch", "TextAction",
        "TimeLeft", "TopProcesses", "TrailingValue", "TrialItemView", "TrialPanel", "TypedField", "UptimeCard", "UptimeCells", "VMsCard",
        "ValueText", "VerbButton", "WorkspacePage", "Wrap",
    ]

    // invariant: every view a file of the kit declares at its top level
    // records itself, by a `.recorded(` in its declaration; the names of
    // those that do not, in their files' order.
    static func unrecorded(_ files: [(path: String, text: String)]) -> [String] {
        var out: [String] = []
        for (path, text) in files where views.contains(where: path.hasPrefix) {
            let lines = text.components(separatedBy: "\n")
            func matches(_ re: NSRegularExpression, _ line: String) -> NSTextCheckingResult? { re.firstMatch(in: line, range: NSRange(line.startIndex..., in: line)) }
            let starts = lines.indices.filter { matches(declaration, lines[$0]) != nil } + [lines.count]
            for (start, end) in zip(starts, starts.dropFirst()) {
                guard let found = matches(view, lines[start]), let name = Range(found.range(at: 1), in: lines[start]) else { continue }
                if !lines[start..<end].contains(where: { $0.contains(".recorded(") }) { out.append(String(lines[start][name])) }
            }
        }
        return out
    }

    func testEveryViewOfTheKitRecordsItselfButThePendingOnes() {
        let files = KitBoundaryTests.swiftFiles(Tree.sources)
        XCTAssertGreaterThan(files.count, 5)
        XCTAssertEqual(Self.unrecorded(files).sorted(), Self.pending)
    }

    func testTheCoverageScanNamesAViewThatRecordsNothing() {
        let found = Self.unrecorded([
            ("atom/Seen.swift", "public struct Seen: View {\n    var body: some View { Text(\"a\").recorded(.text(\"a\", .rowName)) }\n}\n\nstruct Unseen<Label: View>: View {\n    var body: some View { Text(\"b\") }\n}"),
            ("token/Swatch.swift", "struct Swatch: View {\n    var body: some View { Text(\"c\") }\n}"),
        ])
        XCTAssertEqual(found, ["Unseen"])
    }
}
