import SwiftUI

public struct LinkActions {
    public var open: (URL) -> Void
    public var qr: String?
    public var toggleQR: (String) -> Void

    public init(open: @escaping (URL) -> Void = { _ in }, qr: String? = nil, toggleQR: @escaping (String) -> Void = { _ in }) {
        self.open = open
        self.qr = qr
        self.toggleQR = toggleQR
    }
}

public struct LinkView: View {
    let link: Link
    let actions: LinkActions?
    @Environment(\.stationPalette) private var palette

    public init(_ link: Link, actions: LinkActions?) {
        self.link = link
        self.actions = actions
    }

    var key: String { link.label ?? Format.show(link.lead ?? Value()) }

    public var body: some View {
        VStack(alignment: .leading, spacing: Token.Gap.item) {
            HStack(spacing: Token.Space.gap) {
                VStack(alignment: .leading, spacing: Token.Gap.line) {
                    if let lead = link.lead, !lead.isNone {
                        ValueText(lead).fontWeight(.medium)
                    } else {
                        Text(link.label ?? "").font(.system(size: Token.Space.text, weight: .medium)).foregroundStyle(palette.text)
                    }
                    if let value = link.value {
                        Text(Format.show(value)).font(.system(size: Token.Space.small, design: .monospaced)).foregroundStyle(palette.accent).lineLimit(1).truncationMode(.middle)
                    }
                }
                Spacer(minLength: Token.Space.gap)
                if let actions, let url = link.value.flatMap(Format.url) {
                    LinkButton(symbol: Token.Symbol.qr, picked: actions.qr == url.absoluteString) { actions.toggleQR(url.absoluteString) }
                    LinkButton(symbol: Token.Symbol.open, picked: false) { actions.open(url) }
                }
            }
            .padding(.vertical, Token.Gap.inner)
            if let actions, let url = link.value.flatMap(Format.url), actions.qr == url.absoluteString {
                QRView(url.absoluteString).padding(Token.Gap.edge).background(RoundedRectangle(cornerRadius: Token.Tile.qrCorner, style: .continuous).fill(palette.qrLight)).frame(maxWidth: .infinity)
            }
        }
    }
}

// value: a small square button beside a link, 26 pt, 8 pt round.
struct LinkButton: View {
    let symbol: String
    let picked: Bool
    let tap: () -> Void
    @Environment(\.stationPalette) private var palette

    var body: some View {
        Button(action: tap) {
            Image(systemName: symbol).font(.system(size: Token.Space.small + Token.Gap.line, weight: .medium))
                .foregroundStyle(picked ? palette.onTint : palette.text)
                .frame(width: Token.Tile.icon, height: Token.Tile.icon)
                .background(RoundedRectangle(cornerRadius: Token.Tile.hover, style: .continuous).fill(picked ? AnyShapeStyle(palette.accent) : Token.Ink.track))
        }
        .buttonStyle(.plain)
    }
}

public struct ItemsView: View {
    let items: [Item]
    let columns: [Column]
    let width: CGFloat
    let links: LinkActions?
    let hintButtons: Bool

    public init(_ items: [Item], columns: [Column], width: CGFloat, links: LinkActions? = nil, hintButtons: Bool = false) {
        self.items = items
        self.columns = columns
        self.width = width
        self.links = links
        self.hintButtons = hintButtons
    }

    public var body: some View {
        VStack(alignment: .leading, spacing: Token.Gap.tight) {
            ForEach(Array(items.enumerated()), id: \.offset) { _, item in
                ItemView(item: item, columns: columns, width: width, links: links, hintButtons: hintButtons)
            }
        }
    }
}

struct ItemView: View {
    let item: Item
    let columns: [Column]
    let width: CGFloat
    let links: LinkActions?
    let hintButtons: Bool
    @Environment(\.stationPalette) private var palette

    var body: some View {
        if let row = item.row {
            RowView(row, columns: columns, width: width, hintAsButton: hintButtons)
        } else if let fact = item.fact {
            FactView(fact)
        } else if let group = item.group {
            GroupView(group: group, columns: columns, width: width, links: links, hintButtons: hintButtons)
        } else if let empty = item.empty {
            EmptyLine(empty)
        } else if let error = item.error {
            ErrorLine(error)
        } else if let note = item.note {
            Text(note.text ?? "").font(.system(size: Token.Space.small)).foregroundStyle(palette.muted)
        } else if let link = item.link {
            LinkView(link, actions: links)
        }
    }
}

struct GroupView: View {
    let group: Group
    let columns: [Column]
    let width: CGFloat
    let links: LinkActions?
    let hintButtons: Bool

    var children: some View {
        ItemsView(group.items ?? [], columns: columns, width: width - Token.Space.indent, links: links, hintButtons: hintButtons)
            .padding(.leading, Token.Space.indent)
    }

    var body: some View {
        let head = RowView(group.head ?? Row(), columns: columns, width: width, hintAsButton: hintButtons)
        if group.fold == nil || group.fold == "open" {
            VStack(alignment: .leading, spacing: Token.Gap.tight) {
                head
                children
            }
        } else {
            DisclosureGroup { children } label: { head }
        }
    }
}

public struct SectionView: View {
    let section: Section
    let width: CGFloat
    let links: LinkActions?
    let hintButtons: Bool
    let title: String?
    @Environment(\.stationPalette) private var palette

    public init(_ section: Section, width: CGFloat, links: LinkActions? = nil, hintButtons: Bool = false, title: String? = nil) {
        self.section = section
        self.width = width
        self.links = links
        self.hintButtons = hintButtons
        self.title = title
    }

    var head: some View {
        HStack(spacing: Token.Space.gap) {
            Text(title ?? section.title ?? "").font(.system(size: Token.Space.text, weight: .semibold)).foregroundStyle(palette.text)
            ForEach(Array((section.caption ?? []).enumerated()), id: \.offset) { _, c in ValueText(c, muted: true) }
            Spacer(minLength: 0)
            CountsView(section.counts ?? [])
        }
    }

    var items: some View {
        ItemsView(section.items ?? [], columns: section.columns ?? [], width: width, links: links, hintButtons: hintButtons)
    }

    public var body: some View {
        if section.fold == "folded" {
            DisclosureGroup { items } label: { head }
        } else {
            VStack(alignment: .leading, spacing: Token.Gap.small) {
                head
                items
            }
        }
    }
}

public struct StripView: View {
    let strip: Strip
    @Environment(\.stationPalette) private var palette

    public init(_ strip: Strip) { self.strip = strip }

    // invariant: AC-25: CPU, Memory and Network side by side, each its
    // sparkline over its value; the battery and a held Awake on a line
    // below.
    public var body: some View {
        LiveRead { store in drawn(store) }
    }

    func drawn(_ store: LiveStore) -> some View {
        let size = Token.Space.small
        let memorySpark = store.points(strip.memory, else: strip.memory_spark)
        return VStack(alignment: .leading, spacing: Token.Gap.small) {
            HStack(alignment: .top, spacing: Token.Space.gap) {
                measure("CPU", store.points(strip.cpu, else: strip.spark)) { ValueText(store.now(strip.cpu), size: size) }
                if let memory = store.now(strip.memory) {
                    measure("Memory", memorySpark) {
                        if memorySpark?.count ?? 0 < 2, memory.kind == "ratio", let of = memory.of, of > 0 {
                            Meter(Double(memory.n ?? 0) / Double(of))
                        }
                        ValueText(memory, size: size)
                    }
                }
                if let net = strip.net, net.count == 2 {
                    measure("Network", strip.net_spark) {
                        Text("\(Token.Glyph.received)\(Format.show(net[0])) \(Token.Glyph.sent)\(Format.show(net[1]))")
                            .font(.system(size: size)).foregroundStyle(palette.text).monospacedDigit().lineLimit(1).minimumScaleFactor(Token.Detail.figureShrink)
                    }
                }
            }
            if strip.battery != nil || strip.awake.map({ !$0.isNone }) == true {
                HStack(spacing: Token.Space.gap) {
                    if let battery = strip.battery {
                        HStack(spacing: Token.Gap.tight) {
                            Text(Token.Glyph.charging).font(.system(size: size)).foregroundStyle(palette.muted)
                            ValueText(battery, size: size)
                        }
                        .fixedSize()
                    }
                    if let awake = strip.awake, !awake.isNone {
                        Text("\(Token.Glyph.awake) \(Format.show(awake.items?.last ?? awake))")
                            .font(.system(size: size, weight: .semibold)).foregroundStyle(palette.accent)
                            .lineLimit(1).fixedSize()
                    }
                    Spacer(minLength: 0)
                }
            }
        }
    }

    func measure<V: View>(_ label: String, _ spark: [Int]?, @ViewBuilder value: () -> V) -> some View {
        VStack(alignment: .leading, spacing: Token.Gap.tight) {
            Text(label).font(.system(size: Token.Space.small - Token.Gap.line, weight: .semibold)).foregroundStyle(palette.muted)
            if let spark, spark.count > 1 { Spark(spark) }
            HStack(spacing: Token.Gap.small) { value() }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
    }
}

public struct StatusBar: View {
    let line: String
    let failed: Bool
    @Environment(\.stationPalette) private var palette

    public init(_ line: String, failed: Bool) {
        self.line = line
        self.failed = failed
    }

    public var body: some View {
        HStack(spacing: Token.Space.gap / 2) {
            if failed { MarkView(.broken) }
            Text(line).font(.system(size: Token.Space.small)).foregroundStyle(failed ? palette.text : palette.muted).lineLimit(2)
            Spacer(minLength: 0)
        }
    }
}
