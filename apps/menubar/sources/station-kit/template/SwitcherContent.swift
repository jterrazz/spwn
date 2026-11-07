import SwiftUI

// invariant: SPEC §10.2's switcher, vision panel b: a search, the
// machines opened last with This Mac first and checked, every machine
// as Home lists them, each VM with its kind, then Show all machines and
// Add a Mac.
struct SwitcherContent: View {
    let page: Page
    let state: PanelState
    let handlers: PopoverHandlers

    var tree: Section? { page.section("machines") }

    // value: the line lit is one of Recent's (drawn first, so its index is
    // under Recent's count).
    var recentLit: Bool {
        guard let tree, state.search.isEmpty, let lit else { return false }
        return lit < (thisMacRow(tree) == nil ? 0 : 1) + rows(tree, state.recent.filter { $0 != thisMacRow(tree)?.id }).count
    }

    func matching(_ s: Section) -> Set<String>? {
        guard !state.search.isEmpty else { return nil }
        var out = Set<String>()
        func walk(_ items: [Item]) {
            for it in items {
                if let r = it.row ?? it.group?.head, (r.label ?? "").localizedCaseInsensitiveContains(state.search) { out.insert(r.id ?? "") }
                walk(it.group?.items ?? [])
            }
        }
        walk(s.items ?? [])
        return out
    }

    // value: This Mac's line of the tree, where the owner is.
    func thisMacRow(_ s: Section) -> Row? {
        (s.items ?? []).compactMap { $0.group?.head }.first { $0.label == Page.ThisMacWord }
    }

    // value: a VM's line under Recent, as the vision's: its kind, the Mac
    // it runs on, then its health or state; a Mac keeps its own words.
    func recentWords(_ s: Section, _ r: Row) -> String? {
        guard let kind = r.cells?.first?.text, kind != "Mac",
              let mac = (s.items ?? []).compactMap(\.group).first(where: { g in
                  (g.items ?? []).contains { ($0.group?.items ?? []).contains { $0.row?.id == r.id } || $0.row?.id == r.id }
              })?.head?.label else { return nil }
        let state = (r.cells?.last?.items ?? []).dropFirst().first.map { Format.show($0) }
        return (["\(kind) on \(mac)"] + [state].compactMap { $0 }).joined(separator: Token.Glyph.sep)
    }

    func rows(_ s: Section, _ ids: [String]) -> [Row] {
        var out: [Row] = []
        func walk(_ items: [Item]) {
            for it in items {
                if let r = it.row ?? it.group?.head, ids.contains(r.id ?? "") { out.append(r) }
                walk(it.group?.items ?? [])
            }
        }
        walk(s.items ?? [])
        return ids.compactMap { id in out.first { $0.id == id } }
    }

    @State private var lit: Int?

    // value: the lines the keys move through, in the order they are drawn:
    // Recent then the tree, or what the search found.
    func lines(_ tree: Section) -> [Row] {
        var out: [Row] = []
        if state.search.isEmpty {
            if let mac = thisMacRow(tree) { out.append(mac) }
            out += rows(tree, state.recent.filter { $0 != thisMacRow(tree)?.id })
        }
        for line in MachineTree(section: tree, kinds: true, only: matching(tree), open: { _ in }).lines() {
            if case let .row(r, _, _) = line { out.append(r) }
        }
        return out
    }


    func move(_ by: Int) {
        guard let tree else { return }
        let n = lines(tree).count
        guard n > 0 else { return }
        lit = min(max((lit ?? (by > 0 ? -1 : n)) + by, 0), n - 1)
    }

    func submit() {
        guard let tree else { return }
        let all = lines(tree)
        guard let i = SwitcherKeys.picked(all.count, lit: lit, searching: !state.search.isEmpty) else { return }
        let r = all[i]
        if r.id == thisMacRow(tree)?.id, state.search.isEmpty { handlers.back() } else { handlers.openMachine(r.id ?? "") }
    }

    var litID: String? {
        guard let tree else { return nil }
        let all = lines(tree)
        return SwitcherKeys.picked(all.count, lit: lit, searching: !state.search.isEmpty).map { all[$0].id ?? "" }
    }

    var body: some View {
        VStack(alignment: .leading, spacing: Token.Tile.section) {
            TypedField("Search machines", text: Binding(get: { state.search }, set: { lit = nil; handlers.search($0) }), symbol: Token.Symbol.search,
                       autofocus: true, move: move, submit: submit)
            if let tree {
                if state.search.isEmpty {
                    SectionHeader("Recent")
                    Rows {
                        if let mac = thisMacRow(tree) {
                            MachineRow(row: mac, symbol: Token.Symbol.mac, kinds: true, sub: "\(mac.id ?? "")\(Token.Glyph.sep)you are here", picked: true, lit: litID == mac.id && lit != nil, open: { _ in handlers.back() })
                        }
                        ForEach(Array(rows(tree, state.recent.filter { $0 != thisMacRow(tree)?.id }).enumerated()), id: \.offset) { _, r in
                            MachineRow(row: r, symbol: Token.Symbol.machine(r.cells?.first?.text), kinds: true, sub: recentWords(tree, r), lit: litID == r.id && recentLit, open: handlers.openMachine)
                        }
                    }
                }
                SectionHeader(tree.title ?? "")
                MachineTree(section: tree, kinds: true, only: matching(tree), lit: recentLit ? nil : litID, open: handlers.openMachine)
            }
            SheetFor(page: page, state: state, handlers: handlers)
            VStack(alignment: .leading, spacing: 0) {
                SectionDivider().padding(.bottom, Token.Gap.tight)
                Button(action: handlers.back) {
                    MenuText(symbol: Token.Symbol.list, title: "Show all machines", key: Token.Key.shown("cmd+0"), danger: false)
                }
                .buttonStyle(.plain)
                ForEach(Array(page.pageActions.enumerated()), id: \.offset) { _, a in
                    MenuLine(action: a, symbol: Token.Symbol.plus, danger: false) { perform(a, handlers) }
                }
            }
        }
    }
}

public enum SwitcherKeys {
    // invariant: the line lit is the one Return opens; while a search finds
    // something and none is lit, Return opens the first found.
    public static func picked(_ count: Int, lit: Int?, searching: Bool) -> Int? {
        guard count > 0 else { return nil }
        if let lit { return min(max(lit, 0), count - 1) }
        return searching ? 0 : nil
    }
}
