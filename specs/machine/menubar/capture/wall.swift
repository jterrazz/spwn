// value: draws one of M1's three wallpapers (the contrast round's walls.js,
// ported to CoreGraphics) over the guest's whole screen, its scene placed so
// the panel lands where M1 drew it, then sets it as every screen's desktop.
// usage: swift wall.swift <dark|light|color> <out.png> <panel left pt> <panel top pt>
//   the panel's top-left is in points from the screen's top-left corner.
import AppKit

let args = CommandLine.arguments
guard args.count == 5, let left = Double(args[3]), let top = Double(args[4]), let screen = NSScreen.main else {
    FileHandle.standardError.write(Data("usage: wall.swift <dark|light|color> <out.png> <left> <top>\n".utf8))
    exit(2)
}
let kind = args[1]
let scale = screen.backingScaleFactor
let size = screen.frame.size
let width = Int(size.width * scale), height = Int(size.height * scale)
// value: M1's scene is 420 x 452 pt with the panel at (40, 34).
let ox = left - 40, oy = top - 34

struct Rng {
    var s: UInt32
    mutating func next() -> Double {
        s = s &* 1_664_525 &+ 1_013_904_223
        return Double(s) / 4_294_967_296
    }
}

let space = CGColorSpace(name: CGColorSpace.sRGB)!
let g = CGContext(data: nil, width: width, height: height, bitsPerComponent: 8, bytesPerRow: width * 4, space: space, bitmapInfo: CGImageAlphaInfo.premultipliedLast.rawValue)!
g.scaleBy(x: scale, y: scale)
g.translateBy(x: 0, y: size.height)
g.scaleBy(x: 1, y: -1)

func rgba(_ hex: UInt32, _ a: Double = 1) -> CGColor {
    CGColor(srgbRed: Double((hex >> 16) & 0xFF) / 255, green: Double((hex >> 8) & 0xFF) / 255, blue: Double(hex & 0xFF) / 255, alpha: a)
}
func c(_ r: Double, _ gr: Double, _ b: Double, _ a: Double) -> CGColor { CGColor(srgbRed: r / 255, green: gr / 255, blue: b / 255, alpha: a) }
func fill(_ color: CGColor) {
    g.setFillColor(color)
    g.fill(CGRect(origin: .zero, size: size))
}
// value: walls.js draws on a 2x canvas, so its coordinates are halved here.
func blob(_ x: Double, _ y: Double, _ r: Double, _ c0: CGColor, _ c1: CGColor) {
    let gradient = CGGradient(colorsSpace: space, colors: [c0, c1] as CFArray, locations: [0, 1])!
    let p = CGPoint(x: ox + x / 2, y: oy + y / 2)
    g.drawRadialGradient(gradient, startCenter: p, startRadius: 0, endCenter: p, endRadius: r / 2, options: [.drawsAfterEndLocation])
}
func linear(_ stops: [(Double, UInt32)], from: CGPoint, to: CGPoint) {
    let gradient = CGGradient(colorsSpace: space, colors: stops.map { rgba($0.1) } as CFArray, locations: stops.map { CGFloat($0.0) })!
    g.drawLinearGradient(gradient, start: from, end: to, options: [.drawsBeforeStartLocation, .drawsAfterEndLocation])
}
func window(x: Double, y: Double, w: Double, h: Double, bg: UInt32, bar: UInt32, ink: UInt32, lines: Int, bold: Bool, seed: UInt32) {
    let frame = CGRect(x: ox + x, y: oy + y, width: w, height: h)
    let shape = CGPath(roundedRect: frame, cornerWidth: 12, cornerHeight: 12, transform: nil)
    g.saveGState()
    g.setShadow(offset: CGSize(width: 0, height: 10), blur: 30, color: c(0, 0, 0, 0.35))
    g.addPath(shape)
    g.setFillColor(rgba(bg))
    g.fillPath()
    g.restoreGState()
    g.saveGState()
    g.addPath(shape)
    g.clip()
    g.setFillColor(rgba(bar))
    g.fill(CGRect(x: frame.minX, y: frame.minY, width: w, height: 26))
    var r = Rng(s: seed)
    var ly = frame.minY + 26
    for i in 0..<lines {
        let tall = bold && i % 5 == 0
        ly += tall ? 14 : 10
        let lh: Double = tall ? 8 : 3
        let lw = (w - 28) * (40 + r.next() * 55) / 100
        g.setFillColor(rgba(ink))
        g.addPath(CGPath(roundedRect: CGRect(x: frame.minX + 14, y: ly, width: lw, height: lh), cornerWidth: 1.5, cornerHeight: 1.5, transform: nil))
        g.fillPath()
        ly += lh
    }
    g.restoreGState()
}
func noise(_ amount: Double, seed: UInt32) {
    guard let data = g.data else { return }
    let p = data.bindMemory(to: UInt8.self, capacity: width * height * 4)
    var r = Rng(s: seed)
    for i in stride(from: 0, to: width * height * 4, by: 4) {
        let n = Int((r.next() - 0.5) * amount)
        for k in 0..<3 { p[i + k] = UInt8(max(0, min(255, Int(p[i + k]) + n))) }
    }
}

var r = Rng(s: 42)
let area = size.width * size.height / (420 * 452)
switch kind {
case "dark":
    fill(rgba(0x0C0C0E))
    blob(620, 300, 420, c(70, 70, 74, 0.55), c(0, 0, 0, 0))
    blob(150, 120, 300, c(50, 50, 54, 0.5), c(0, 0, 0, 0))
    noise(26, seed: 42)
    for _ in 0..<Int(900 * area) {
        let b = 120 + r.next() * 135, s = r.next() < 0.06 ? 1.2 : r.next() < 0.3 ? 0.7 : 0.4
        g.setFillColor(c(b, b, b, 0.4 + r.next() * 0.6))
        g.fillEllipse(in: CGRect(x: r.next() * size.width - s, y: r.next() * size.height - s, width: 2 * s, height: 2 * s))
    }
    for _ in 0..<Int(40 * area) {
        let x = r.next() * size.width, y = r.next() * size.height
        g.setStrokeColor(c(230, 230, 230, 0.5))
        g.setLineWidth(0.6)
        g.move(to: CGPoint(x: x, y: y))
        g.addLine(to: CGPoint(x: x + 3, y: y - 1.5))
        g.strokePath()
    }
    window(x: -30, y: 210, w: 225, h: 260, bg: 0xFFFFFF, bar: 0xECECEC, ink: 0x555555, lines: 14, bold: true, seed: 3)
    window(x: 205, y: 395, w: 260, h: 90, bg: 0x3A3A3C, bar: 0x2C2C2E, ink: 0x8E8E93, lines: 5, bold: false, seed: 9)
    g.setFillColor(rgba(0xFF7A2F))
    g.fillEllipse(in: CGRect(x: ox + 250, y: oy + 300, width: 34, height: 34))
case "light":
    linear([(0, 0xA9D4F7), (0.55, 0xF3EFE6), (1, 0xE9D9B8)], from: CGPoint(x: 0, y: oy), to: CGPoint(x: 0, y: oy + 452))
    blob(680, 120, 260, c(255, 255, 240, 1), c(255, 255, 240, 0))
    blob(120, 700, 330, c(120, 170, 90, 0.55), c(120, 170, 90, 0))
    blob(760, 760, 280, c(240, 160, 80, 0.45), c(240, 160, 80, 0))
    var s = Rng(s: 5)
    for _ in 0..<Int(160 * area) {
        g.setFillColor(c(255, 255, 255, s.next() * 0.5))
        let x = s.next() * size.width, y = s.next() * size.height, rad = (4 + s.next() * 30) / 2
        g.fillEllipse(in: CGRect(x: x - rad, y: y - rad, width: 2 * rad, height: 2 * rad))
    }
    window(x: -20, y: 150, w: 240, h: 330, bg: 0xFFFFFF, bar: 0xF2F2F2, ink: 0x333333, lines: 18, bold: true, seed: 11)
    window(x: 230, y: 250, w: 230, h: 230, bg: 0xFAFAFA, bar: 0xE6E6E6, ink: 0x555555, lines: 14, bold: false, seed: 4)
case "color":
    linear([(0, 0x1747D1), (0.5, 0x4B3FE0), (1, 0x8A3BE6)], from: CGPoint(x: ox, y: oy), to: CGPoint(x: ox + 420, y: oy + 452))
    blob(160, 200, 420, c(60, 200, 255, 0.85), c(60, 200, 255, 0))
    blob(720, 650, 420, c(230, 80, 255, 0.75), c(230, 80, 255, 0))
    blob(520, 260, 260, c(255, 255, 255, 0.35), c(255, 255, 255, 0))
    blob(300, 820, 300, c(10, 20, 90, 0.8), c(10, 20, 90, 0))
default:
    FileHandle.standardError.write(Data("no wall \(kind)\n".utf8))
    exit(2)
}

let url = URL(fileURLWithPath: args[2])
let image = g.makeImage()!
let dest = CGImageDestinationCreateWithURL(url as CFURL, "public.png" as CFString, 1, nil)!
CGImageDestinationAddImage(dest, image, nil)
guard CGImageDestinationFinalize(dest) else { exit(1) }
do {
    for s in NSScreen.screens { try NSWorkspace.shared.setDesktopImageURL(url, for: s, options: [:]) }
} catch {
    FileHandle.standardError.write(Data("setDesktopImageURL: \(error.localizedDescription)\n".utf8))
    exit(1)
}
print("wall \(kind) \(width)x\(height) at \(url.path)")
