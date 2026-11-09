import SwiftUI

// invariant: SPEC A6: Find in the panel's place, in the same glass: its
// field, the results of FindIndex grouped under their labels, or the
// no-match lines, then the footer bar with esc Close and the verbs of the
// line lit. Running a result closes Find; an action that asks first opens
// its sheet over Find; "Show N more" opens its group in place.
struct FindPanel: View {
    let state: PanelState
    let handlers: PopoverHandlers
    let scrolls: Bool
    @Environment(\.stationTokens) private var tokens

    var find: FindQuery { state.find ?? FindQuery() }
    var index: Page { state.findIndex ?? Page() }
    var groups: [FindGroup] { FindIndex(index).results(query: find.query, recent: state.recent, open: Set(find.open)) }
    var lines: [FindLine] { FindIndex.lines(groups) }

    var lit: FindLine? {
        let all = lines
        return all.isEmpty ? nil : all[min(max(find.selected, 0), all.count - 1)]
    }

    var body: some View {
        Panel(insets: .flush, bounded: scrolls, key: "\(state.openings)/find", sheet: state.confirming) {
            FindField(text: Binding(get: { find.query }, set: { handlers.find(find.typed($0)) }),
                      move: { handlers.find(find.moved($0, lines: lines.count)) },
                      submit: { quick in if let lit { pick(lit, quick: quick) } },
                      close: { handlers.find(nil) })
                .padding(Token.Metric.Find.margin)
        } content: {
            VStack(alignment: .leading, spacing: 0) {
                if lines.isEmpty, !find.query.trimmingCharacters(in: .whitespaces).isEmpty {
                    nomatch
                } else {
                    results
                }
                if let id = state.confirming, index.action(id) != nil {
                    SheetFor(page: index, state: state, handlers: handlers).padding(.horizontal, Token.Space.margin)
                }
            }
        } footer: {
            footer
        }
        .background { FindEscape(close: { handlers.find(nil) }) }
        .environment(\.stationHints, false)
    }

    var results: some View {
        let all = lines
        let selected = lit
        return VStack(alignment: .leading, spacing: 0) {
            ForEach(groups, id: \.id) { g in
                Text(g.title)
                    .font(Token.Role.findGroup.font)
                    .foregroundStyle(tokens.color.t3)
                    .frame(height: Token.Find.groupLine)
                    .padding(Token.Find.group)
                ForEach(Array(g.entries.enumerated()), id: \.offset) { _, e in
                    let line = FindLine.entry(e)
                    Button { pick(line, quick: false) } label: {
                        FindResult(e, query: find.query, lit: selected == line, asks: verb(e)?.confirm != nil, key: key(e))
                    }
                    .buttonStyle(.plain)
                    .accessibilityHint(Text(verb(e)?.label ?? ""))
                }
                if g.more > 0 {
                    let line = FindLine.more(group: g.id, count: g.more)
                    Button { pick(line, quick: false) } label: { FindMore(g.more, lit: selected == line) }
                        .buttonStyle(.plain)
                }
            }
        }
        .padding(Token.Find.results)
        .id(all.count)
    }

    var nomatch: some View {
        VStack(spacing: Token.Find.emptySpacing) {
            Text(Token.Find.noMatch(find.query.trimmingCharacters(in: .whitespaces)))
            Text(Token.Find.hint)
        }
        .font(Token.Role.findResult.font)
        .foregroundStyle(tokens.color.t2)
        .multilineTextAlignment(.center)
        .lineSpacing(Token.Find.emptySpacing)
        .frame(maxWidth: .infinity)
        .padding(Token.Find.empty)
        .padding(Token.Find.results)
    }

    var footer: some View {
        FindFooter(verb: footerVerb, alt: footerAlt)
    }

    var footerVerb: String? {
        switch lit {
        case let .entry(e): verb(e)?.label ?? Token.Find.open
        case .more: Token.Find.show
        case nil: nil
        }
    }

    var footerAlt: String? {
        guard case let .entry(e) = lit, let quick = e.row.quick else { return nil }
        return index.action(quick)?.label
    }

    func verb(_ e: FindEntry) -> Action? { e.row.verb.flatMap { index.action($0) } }

    func key(_ e: FindEntry) -> String? {
        guard e.row.quick == nil, let key = verb(e)?.key, !key.isEmpty else { return nil }
        return key
    }

    // invariant: a line picked runs as its row would: "Show N more" opens
    // its group; ⌘↩ runs the alternate verb; a verb that runs a call runs
    // it (its sheet first when it asks); else the page the row opens.
    func pick(_ line: FindLine, quick: Bool) {
        switch line {
        case let .more(group, _):
            handlers.find(find.opening(group))
        case let .entry(e):
            if quick, let id = e.row.quick, let alt = index.action(id) {
                handlers.find(nil)
                perform(alt, handlers)
                return
            }
            if let action = verb(e), action.run != nil {
                if action.confirm != nil || action.form != nil {
                    handlers.confirm(action)
                } else {
                    handlers.find(nil)
                    handlers.run(action, [:])
                }
                return
            }
            handlers.find(nil)
            if let nav = e.row.nav { go(nav, handlers) }
        }
    }
}

// value: Find's footer bar: esc Close, then the verbs of the line lit, the
// alternate after a hairline.
struct FindFooter: View {
    let verb: String?
    let alt: String?
    @Environment(\.stationTokens) private var tokens

    var body: some View {
        let f = Token.Find.self
        HStack(spacing: f.footerGap) {
            KeyCap(f.closeKey)
            Text(f.close).foregroundStyle(tokens.color.t3)
            Spacer(minLength: 0)
            if let verb {
                Text(verb)
                KeyCap(f.runKey)
            }
            if let alt {
                Rectangle().fill(tokens.color.hair).frame(width: f.divider.width, height: f.divider.height).padding(.horizontal, f.dividerMargin)
                Text(alt)
                KeyCap(f.altKey)
            }
        }
        .font(.system(size: f.footerText))
        .foregroundStyle(tokens.color.t2)
        .lineLimit(1)
        .padding(.horizontal, f.footerPadding)
        .frame(height: Token.Metric.Find.footer)
        .overlay(alignment: .top) { Rectangle().fill(tokens.color.hair).frame(height: Token.Metric.Row.hairline) }
        .padding(.top, f.footerTop)
    }
}

// value: esc closes Find, never the panel under it.
struct FindEscape: View {
    let close: () -> Void
    @Environment(\.stationStill) private var still

    var body: some View {
        if !still {
            Button("", action: close)
                .keyboardShortcut(.escape, modifiers: [])
                .opacity(0)
                .allowsHitTesting(false)
                .accessibilityHidden(true)
        }
    }
}
