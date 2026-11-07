import Charts
import CoreImage.CIFilterBuiltins
import SwiftUI

public struct MarkView: View {
    let mark: Mark?
    @Environment(\.stationPalette) private var palette

    public init(_ mark: Mark?) { self.mark = mark }

    public var body: some View {
        if let mark {
            if mark == .pending {
                PendingDot()
            } else {
                Image(systemName: Token.Symbol.of(mark))
                    .font(.system(size: Token.Space.mark))
                    .foregroundStyle(palette.of(mark))
                    .frame(width: Token.Space.mark + Token.Gap.tight)
            }
        }
    }
}

public struct PendingDot: View {
    @Environment(\.stationPalette) private var palette
    @Environment(\.stationStill) private var frozen
    let pinned: Bool

    public init(still: Bool = false) { pinned = still }

    var still: Bool { pinned || frozen }

    public var body: some View {
        Image(systemName: Token.Symbol.pending)
            .font(.system(size: Token.Space.mark))
            .foregroundStyle(palette.muted)
            .symbolEffect(.rotate, isActive: !still)
            .symbolEffect(.pulse, isActive: !still)
            .frame(width: Token.Space.mark + Token.Gap.tight)
    }
}

public struct ValueText: View {
    let value: Value?
    let muted: Bool
    let size: CGFloat
    @Environment(\.stationPalette) private var palette
    @Environment(\.stationNow) private var now

    public init(_ value: Value?, muted: Bool = false, size: CGFloat = Token.Space.text) {
        self.value = value
        self.muted = muted
        self.size = size
    }

    public var body: some View {
        if let value {
            if value.kind == "pending" {
                PendingDot()
            } else {
                let text = Text(Format.show(value, now: now))
                    .font(.system(size: size))
                    .contentTransition(.numericText())
                    .foregroundStyle(muted || value.kind == "command" || value.stale == true ? palette.muted : palette.text)
                    .lineLimit(1)
                    .truncationMode(value.kind == "path" ? .middle : .tail)
                if Format.numeric(value) { text.monospacedDigit() } else { text }
            }
        }
    }
}

public struct Meter: View {
    let fraction: Double
    @Environment(\.stationPalette) private var palette

    public init(_ fraction: Double) { self.fraction = min(max(fraction, 0), 1) }

    public var body: some View {
        ZStack(alignment: .leading) {
            Capsule().fill(Token.Ink.track)
            Capsule().fill(palette.accent).frame(width: Token.Space.meterWidth * fraction)
        }
        .frame(width: Token.Space.meterWidth, height: Token.Space.meterHeight)
    }
}

public enum QR {
    public static func modules(_ text: String) -> CGImage? {
        let filter = CIFilter.qrCodeGenerator()
        filter.message = Data(text.utf8)
        filter.correctionLevel = "L"
        guard let image = filter.outputImage else { return nil }
        return CIContext().createCGImage(image, from: image.extent)
    }
}

public struct QRView: View {
    let text: String
    @Environment(\.stationPalette) private var palette

    public init(_ text: String) { self.text = text }

    public var body: some View {
        if let image = QR.modules(text) {
            Image(decorative: image, scale: 1)
                .interpolation(.none)
                .resizable()
                .aspectRatio(1, contentMode: .fit)
                .frame(width: Token.Tile.qr, height: Token.Tile.qr)
        }
    }
}

// value: a menu item's glyph: the vision's own `>_` prompt and station
// mark, drawn as its 16-unit strokes at 14 pt; any other name is an SF
// Symbol at 13 pt.
public struct MenuGlyph: View {
    let symbol: String

    public init(_ symbol: String) { self.symbol = symbol }

    public var body: some View {
        switch symbol {
        case Token.Symbol.prompt, Token.Symbol.list:
            Stroke16(symbol: symbol)
                .stroke(style: StrokeStyle(lineWidth: Token.Stroke.graph, lineCap: .round, lineJoin: .round))
                .frame(width: Token.Detail.menuStroke, height: Token.Detail.menuStroke)
        default:
            Image(systemName: symbol).font(.system(size: Token.Detail.menu, weight: .medium))
        }
    }
}

struct Stroke16: Shape {
    let symbol: String

    func path(in rect: CGRect) -> Path {
        let k = rect.width / 16
        func p(_ x: CGFloat, _ y: CGFloat) -> CGPoint { CGPoint(x: rect.minX + x * k, y: rect.minY + y * k) }
        var path = Path()
        if symbol == Token.Symbol.prompt {
            path.move(to: p(3, 4.5))
            path.addLine(to: p(6.5, 8))
            path.addLine(to: p(3, 11.5))
            path.move(to: p(8.5, 12))
            path.addLine(to: p(13, 12))
        } else {
            let inset = Token.Detail.stroke16Inset, box = Token.Detail.stroke16Box * k, corner = Token.Detail.stroke16Corner * k
            path.addRoundedRect(in: CGRect(origin: p(inset, inset), size: CGSize(width: box, height: box)), cornerSize: CGSize(width: corner, height: corner))
            path.move(to: p(5, 6.3))
            path.addLine(to: p(11, 6.3))
            path.move(to: p(5, 9.7))
            path.addLine(to: p(8.6, 9.7))
        }
        return path
    }
}
