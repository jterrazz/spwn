import AppKit
import Foundation
import StationKit
import SwiftUI
import XCTest

@MainActor
final class MenuTests: XCTestCase {
    func menuStories() throws -> [Story] { try Stories.load(root: Tree.stories).filter { $0.group == .menu } }

    // value: a menu as the owner reads it: its header, then each entry's
    // words ("-" between groups), red ones marked, a submenu's in brackets.
    static func read(_ model: MenuModel) -> [String] {
        (model.header.map { [$0] } ?? []) + model.items.map(read)
    }

    static func read(_ item: MenuItem) -> String {
        if item.separator { return "-" }
        var words = (item.checked ? "✓ " : "") + item.title + (item.destructive ? " (red)" : "")
        if let key = item.key, item.submenu.isEmpty { words += " " + MenuModel.keys(key) }
        if !item.submenu.isEmpty { words += " [" + item.submenu.map(read).joined(separator: ", ") + "]" }
        if let note = item.note { words += " · " + note + (item.enabled ? "" : " (off)") }
        return words
    }

    // value: the menu the panel attaches natively where a story opens
    // one: the page's More menu (Home's the station menu), or the menu of
    // the row or the tile right-clicked, found on the page as drawn.
    static func attached(_ story: Story) -> MenuModel? {
        guard let open = story.ui.menu, let page = story.state.shown else { return nil }
        if let sheet = story.ui.sheet, let action = page.action(sheet.id), let form = action.form {
            let answers = Answers(form, sheet.answers ?? [:])
            if open.id == SheetAnchor.split { return MenuModel.split(action, page: page) }
            return (form.fields ?? []).first { SheetAnchor.field($0.key ?? "") == open.id }.map { MenuModel.popup($0, answers) }
        }
        guard let key = open.row else {
            return page.id == PageID.home ? MenuModel.station(page, loginItemOn: story.ui.loginItem ?? false, sparklineOn: false) : MenuModel.page(page)
        }
        let actions = page.actions ?? []
        let rows = (page.sections ?? []).flatMap { s in (s.items ?? []).flatMap { [$0.row].compactMap { $0 } + ($0.group?.items ?? []).compactMap(\.row) } }
        if let row = rows.first(where: { $0.id == key }) { return MenuModel.context(row, actions: actions) }
        let tiles = (page.sections ?? []).flatMap { ($0.items ?? []).compactMap(\.tile) }
        return tiles.first { t in actions.first { $0.id == t.toggle }?.rows?.first == key }.map { MenuModel.tile($0, actions: actions) }
    }

    // invariant: SPEC C4: a menu story draws the very model the panel
    // hands the system, so its golden shows what the native menu lists.
    func testEveryMenuStoryPreviewsTheMenuThePanelAttaches() throws {
        let stories = try menuStories()
        XCTAssertEqual(stories.count, 21)
        for story in stories {
            let drawn = try XCTUnwrap(story.menu, story.id)
            XCTAssertFalse(drawn.isEmpty, story.id)
            XCTAssertEqual(drawn, Self.attached(story), story.id)
        }
    }

    // invariant: SPEC A5's table: each menu lists what its row or page
    // offers, in groups, red what destroys, each key as the system says it.
    func testEachMenuSaysWhatSpecA5Lists() throws {
        let read = Dictionary(uniqueKeysWithValues: try menuStories().compactMap { s in s.menu.map { (s.id, Self.read($0)) } })
        XCTAssertEqual(read["menu.station"], ["New VM… ⌘N", "Add a Mac…", "Open station in Terminal ⌘T", "-", "✓ Open at login", "CPU in the menu bar", "-", "Quit station ⌘Q"])
        XCTAssertEqual(read["menu.context.workspace"], ["linux-1 on This Mac", "Open ↩", "Open in Terminal ⌘↩", "Open Vite", "-",
                                                        "Copy [SSH command, Link to Vite, Link to Port 3000]", "Share [Show QR…, A port…, An account…]", "-",
                                                        "Stop…", "-", "Delete… (red)"])
        XCTAssertEqual(read["menu.new-vm.on"], ["This Mac · 28 GB free", "✓ mac-mini · 24 GB free", "macbook-air · asleep (off)"])
        XCTAssertEqual(read["menu.new-vm.create"], ["Create ↩", "Create and Open in Terminal ⌥↩"])
        let workspaces = ["✓ linux-1 · on This Mac", "agent-7f · on This Mac", "linux-3 · on mac-mini", "review-bot · on mac-mini"]
        XCTAssertEqual(read["menu.lend.to"], workspaces)
        XCTAssertEqual(read["menu.share-port.of"], workspaces)
        XCTAssertEqual(read["menu.context.stopped"], ["api on This Mac", "Open ↩", "-", "Start", "-", "Delete… (red)"])
        XCTAssertEqual(read["menu.context.server.down"]?.contains("Restart"), true)
        XCTAssertEqual(read["menu.context.mac"], ["mac-mini", "Open ↩", "New VM on mac-mini… ⌘N", "Open in Terminal ⌘T"])
        XCTAssertEqual(read["menu.context.mac.asleep"], ["macbook-air", "Open ↩", "Wake macbook-air"])
        XCTAssertEqual(read["menu.context.lease"], ["GitHub lent to linux-1", "Extend 4 hr", "-", "Take back… (red)"])
        XCTAssertEqual(read["menu.awake"], ["Keep This Mac awake", "For 1 hr", "For 4 hr", "✓ Until 18:00", "-", "Let it sleep", "Awake of all Macs ⌘E"])
        XCTAssertEqual(read["menu.page.mac"], ["mac-mini", "Open in Terminal ⌘T", "-", "Restart…", "Stop everything… (red)", "-", "Forget… (red)"])
        XCTAssertEqual(read["menu.page.this-mac"], ["This Mac", "Open in Terminal ⌘T", "-", "Stop everything… (red)"])
        XCTAssertEqual(read["menu.page.mac.asleep"], ["macbook-air", "Wake macbook-air", "-", "Forget… (red)"])
        XCTAssertEqual(read["menu.page.server"], ["infra on mac-mini", "Restart…", "Stop… (red)", "-", "Delete… (red)"])
        XCTAssertEqual(read["menu.page.ports"], ["Share a port…", "Declare a port public…"])
    }

    // invariant: a key reads as the system's menus write it, and the
    // native item answers the same key.
    func testAKeyReadsAndAnswersAsTheSystemsMenus() {
        XCTAssertEqual(["cmd+n", "shift+cmd+n", "cmd+enter", "enter", "cmd+q"].map(MenuModel.keys), ["⌘N", "⇧⌘N", "⌘↩", "↩", "⌘Q"])
        XCTAssertEqual(MenuModel.shortcut("cmd+enter"), KeyboardShortcut(.return, modifiers: .command))
        XCTAssertEqual(MenuModel.shortcut("shift+cmd+n"), KeyboardShortcut("n", modifiers: [.shift, .command]))
        XCTAssertNil(MenuModel.shortcut("hyper+x"))
    }

    // invariant: picking an entry does what it says: a link opens or is
    // copied, Open goes to the row's page, a verb that asks first opens
    // its question, the app's own switches flip.
    func testPickingAnEntryDoesWhatItSays() throws {
        var did: [String] = []
        var h = PopoverHandlers(confirm: { did.append("asks " + ($0?.id ?? "")) }, open: { did.append("opens " + $0.absoluteString) },
                                loginItem: { did.append("login \($0)") }, loginItemOn: true, quit: { did.append("quits") },
                                copy: { did.append("copies " + $0) })
        h.push = { did.append("goes to " + $0) }
        let pick = MenuPick(h)
        let page = try XCTUnwrap(try Stories.load(root: Tree.stories).first { $0.id == "menu.context.workspace" }?.state.shown)
        let model = MenuModel.context(try XCTUnwrap(page.sections?.flatMap { $0.items ?? [] }.compactMap(\.row).first { $0.id == "linux-1" }), actions: page.actions ?? [])
        let entries = model.items + model.items.flatMap(\.submenu)
        for title in ["Open", "Open Vite", "SSH command", "Stop…"] {
            pick(try XCTUnwrap(entries.first { $0.title == title }, title), nav: "menubar.machine:linux-1")
        }
        let home = try XCTUnwrap(try Stories.load(root: Tree.stories).first { $0.id == "menu.station" }?.state.shown)
        for item in MenuModel.station(home, loginItemOn: true, sparklineOn: false).items where item.id == MenuModel.login || item.id == MenuModel.quit {
            pick(item, nav: nil)
        }
        XCTAssertEqual(did, ["goes to menubar.machine:linux-1", "opens https://macbook-pro.tail77a797.ts.net:40001", "copies ssh linux-1", "asks stop@linux-1", "login false", "quits"])
    }

    // value: a native menu as the owner reads it: its header, each entry's
    // words, red ones marked, its key as the system draws it, checked ones
    // ticked, a submenu's in brackets, a popup's grey note after a dot.
    static func read(_ menu: NSMenu) -> [String] {
        menu.items.map { entry in
            if entry.isSeparatorItem { return "-" }
            if entry.isSectionHeader { return entry.title }
            let words = entry.attributedTitle?.string ?? entry.title
            let parts = words.components(separatedBy: "\t")
            var said = (entry.state == .on ? "✓ " : "") + parts[0]
            let ink = entry.attributedTitle?.attribute(.foregroundColor, at: 0, effectiveRange: nil) as? NSColor
            if ink == Token.SystemMenu.destructive { said += " (red)" }
            if !entry.keyEquivalent.isEmpty {
                let m = entry.keyEquivalentModifierMask
                let mods = (m.contains(.option) ? "opt+" : "") + (m.contains(.shift) ? "shift+" : "") + (m.contains(.command) ? "cmd+" : "")
                said += " " + MenuModel.keys(mods + (entry.keyEquivalent == "\r" ? "enter" : entry.keyEquivalent))
            }
            if let sub = entry.submenu { said += " [" + read(sub).joined(separator: ", ") + "]" }
            if parts.count > 1 { said += " · " + parts[1] + (entry.isEnabled ? "" : " (off)") }
            if entry.image != nil { said += " (icon)" }
            return said
        }
    }

    // invariant: SPEC A5: the menu the system shows is the one the story
    // draws: the same header and entries, red what destroys, the keys, the
    // checks, a popup's notes in their grey column, a split button's icons.
    func testTheSystemsMenuSaysWhatTheStoryDraws() throws {
        let stories = try menuStories()
        let shown = Dictionary(uniqueKeysWithValues: stories.compactMap { s in s.menu.map { (s.id, Self.read(SystemMenu.make($0) { _ in })) } })
        for story in stories {
            let drawn = try XCTUnwrap(story.menu, story.id)
            XCTAssertEqual(shown[story.id]?.map { $0.replacingOccurrences(of: " (icon)", with: "") }, Self.read(drawn), story.id)
        }
        XCTAssertEqual(shown["menu.page.mac"]?.map { $0.replacingOccurrences(of: " (icon)", with: "") }, ["mac-mini", "Open in Terminal ⌘T", "-", "Restart…", "Stop everything… (red)", "-", "Forget… (red)"])
        XCTAssertEqual(shown["menu.new-vm.on"], ["This Mac · 28 GB free", "✓ mac-mini · 24 GB free", "macbook-air · asleep (off)"])
        XCTAssertEqual(shown["menu.new-vm.create"], ["Create ↩ (icon)", "Create and Open in Terminal ⌥↩ (icon)"])
    }

    // invariant: a button's menu hangs right-aligned under it, inside the
    // panel; a popup lays its picked entry over its field.
    func testAMenuHangsWhereThePanelDrawsIt() {
        let button = CGRect(x: 280, y: 0, width: 28, height: 28)
        let under = SystemMenu.origin(.under, menu: NSSize(width: 220, height: 160), in: button)
        XCTAssertEqual(under.x + 220, button.maxX)
        XCTAssertEqual(under.y, button.maxY + Token.SystemMenu.drop)
        XCTAssertEqual(SystemMenu.origin(.over, menu: NSSize(width: 220, height: 80), in: CGRect(x: 0, y: 0, width: 180, height: 26)), .zero)
    }
}
