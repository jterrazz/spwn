import CoreGraphics
import Foundation

// usage: swift scripts/symbols.swift <symbols.xcassets>, run by
// build.sh, `make menubar-check` and `make catalog` before they build
// value: the Orbit filled as SF Symbols templates, one per state, drawn on
// the template's 100 pt grid against the system's `circle`: its ring has
// the circle's outer edge and a stroke each master takes heavier than the
// circle's, so a weight and a point size draw the Orbit as they draw a
// filled system symbol.

struct Master {
    let name: String
    let stroke: CGFloat
    let outer: CGFloat
    let share: CGFloat
    let bearing: CGFloat
    let column: CGFloat
    let baseline: CGFloat

    var thick: CGFloat { share * outer }
    var ring: CGFloat { outer - thick / 2 }
    var unit: CGFloat { ring / 7 }
    var bolder: CGFloat { (thick - 2.1 * unit) / 2 }
}

let weights: [(name: String, column: CGFloat, share: CGFloat)] = [("Ultralight", 560, 0.1), ("Regular", 1460, 0.29), ("Black", 2400, 0.39)]
let sizes: [(name: String, baseline: CGFloat, bearing: CGFloat, stroke: [CGFloat], outer: [CGFloat])] = [
    ("S", 696, 7, [2.2, 7.48, 14.17], [37, 39.5, 42]),
    ("M", 1126, 9, [2.32, 8.23, 17.24], [47, 49.5, 53]),
    ("L", 1556, 11, [2.47, 9.16, 20.32], [61.5, 64, 68.5]),
]
let masters = sizes.flatMap { size in
    weights.enumerated().map { i, w in
        Master(name: "\(w.name)-\(size.name)", stroke: size.stroke[i], outer: size.outer[i], share: w.share, bearing: size.bearing, column: w.column, baseline: size.baseline)
    }
}

let capHeight: CGFloat = 70.459

enum State: String, CaseIterable {
    case plain = "station.orbit"
    case badge = "station.orbit.badge"
    case slash = "station.orbit.slash"
    case working = "station.orbit.working"
}

// invariant: every master of a state is drawn by the same commands in the
// same order, never by a boolean of shapes, so the weights interpolate.

func polar(_ degrees: CGFloat, _ r: CGFloat, centre: CGPoint = .zero) -> CGPoint {
    CGPoint(x: centre.x + r * cos(degrees * .pi / 180), y: centre.y + r * sin(degrees * .pi / 180))
}

func degrees(_ p: CGPoint) -> CGFloat { atan2(p.y, p.x) * 180 / .pi }

func arc(_ p: CGMutablePath, _ r: CGFloat, _ from: CGFloat, _ to: CGFloat, centre: CGPoint = .zero, pieces: Int = 4) {
    let step = (to - from) / CGFloat(pieces)
    let k = 4 / 3 * tan(step * .pi / 720) * r
    for i in 0..<pieces {
        let a = (from + CGFloat(i) * step) * .pi / 180, b = (from + CGFloat(i + 1) * step) * .pi / 180
        let p0 = CGPoint(x: centre.x + r * cos(a), y: centre.y + r * sin(a))
        let p3 = CGPoint(x: centre.x + r * cos(b), y: centre.y + r * sin(b))
        p.addCurve(to: p3, control1: CGPoint(x: p0.x - k * sin(a), y: p0.y + k * cos(a)), control2: CGPoint(x: p3.x + k * sin(b), y: p3.y - k * cos(b)))
    }
}

func circle(_ p: CGMutablePath, _ c: CGPoint, _ r: CGFloat) {
    p.move(to: CGPoint(x: c.x + r, y: c.y))
    arc(p, r, 0, 360, centre: c)
    p.closeSubpath()
}

func roundedSquare(_ p: CGMutablePath, half: CGFloat, radius: CGFloat) {
    let c = half - radius
    p.move(to: CGPoint(x: half, y: -c))
    for (centre, a) in [(CGPoint(x: c, y: c), 0), (CGPoint(x: -c, y: c), 90), (CGPoint(x: -c, y: -c), 180), (CGPoint(x: c, y: -c), 270)] as [(CGPoint, CGFloat)] {
        p.addLine(to: polar(a, radius, centre: centre))
        arc(p, radius, a, a + 90, centre: centre, pieces: 1)
    }
    p.closeSubpath()
}

// value: where a piece of the ring ends: against a disc cut out around a
// point of the ring's plane, or against one edge of the slash's gap.
enum End {
    case disc(CGPoint, CGFloat)
    case band(CGFloat)
}

// value: the drawing of concept A on its 18-unit grid, a unit being a
// seventh of the ring's radius: the ring cut around two machines, the
// solid Mac inside it, the two machines on it; needs you cuts a badge out
// at the top right, down cuts a slash across, working moves the machines
// a third of a turn along the ring.
struct Geometry {
    let m: Master
    var u: CGFloat { m.unit }
    var b: CGFloat { m.bolder }
    var ro: CGFloat { m.ring + m.thick / 2 }
    var ri: CGFloat { m.ring - m.thick / 2 }
    var satellite: CGFloat { 2.3 * u + b }
    var clearance: CGFloat { satellite + 1.55 * u }
    var coreHalf: CGFloat { 3.6 * u + b }
    var coreRadius: CGFloat { 2.1 * u + b }
    var coreReach: CGFloat { (coreHalf - coreRadius) * 2.squareRoot() + coreRadius }
    var dotAt: CGPoint { CGPoint(x: 5.6 * u, y: 5.6 * u) }
    var dot: CGFloat { 2.75 * u + b }
    var notch: CGFloat { min(dot + 1.45 * u, hypot(dotAt.x, dotAt.y) - coreReach - 0.15 * u) }
    let slash = CGPoint(x: -6.4 / hypot(6.4, 7.4), y: 7.4 / hypot(6.4, 7.4))
    var normal: CGPoint { CGPoint(x: slash.y, y: -slash.x) }
    var length: CGFloat { hypot(6.4, 7.4) * u }
    var band: CGFloat { m.thick / 2 + 1.6 * u }

    // value: the angle on a circle of radius r around the centre where a
    // piece of the ring meets its end, the piece lying past it on the side
    // the sign gives: +1 counterclockwise, -1 clockwise.
    func angle(_ end: End, _ r: CGFloat, _ sign: CGFloat) -> CGFloat {
        switch end {
        case let .disc(c, radius):
            let d = hypot(c.x, c.y)
            return degrees(c) + sign * acos((r * r + d * d - radius * radius) / (2 * r * d)) * 180 / .pi
        case let .band(side):
            return degrees(CGPoint(x: side * normal.x, y: side * normal.y)) - sign * acos(band / r) * 180 / .pi
        }
    }

    func edge(_ p: CGMutablePath, _ end: End, to: CGPoint) {
        guard case let .disc(c, radius) = end else { return p.addLine(to: to) }
        let from = degrees(CGPoint(x: p.currentPoint.x - c.x, y: p.currentPoint.y - c.y))
        var delta = degrees(CGPoint(x: to.x - c.x, y: to.y - c.y)) - from
        while delta > 180 { delta -= 360 }
        while delta <= -180 { delta += 360 }
        arc(p, radius, from, from + delta, centre: c, pieces: 1)
    }

    func piece(_ p: CGMutablePath, from start: End, to end: End) {
        let os = angle(start, ro, 1), isn = angle(start, ri, 1)
        var oe = angle(end, ro, -1), ie = angle(end, ri, -1)
        while oe <= os { oe += 360 }
        while ie <= isn { ie += 360 }
        p.move(to: polar(os, ro))
        arc(p, ro, os, oe)
        edge(p, end, to: polar(ie, ri))
        arc(p, ri, ie, isn)
        edge(p, start, to: polar(os, ro))
        p.closeSubpath()
    }

    // value: the Mac cut by the slash's gap into its two corners away from
    // the slash, each from the gap's edge along its top round to its side.
    func slashedCore(_ p: CGMutablePath) {
        let h = coreHalf, c = coreHalf - coreRadius
        let top = (band - normal.y * h) / normal.x, side = (band - normal.x * h) / normal.y
        for sign in [1, -1] as [CGFloat] {
            func pt(_ x: CGFloat, _ y: CGFloat) -> CGPoint { CGPoint(x: sign * x, y: sign * y) }
            let turn: CGFloat = sign > 0 ? 0 : 180
            p.move(to: pt(top, h))
            p.addLine(to: pt(c, h))
            arc(p, coreRadius, 90 + turn, turn, centre: pt(c, c), pieces: 1)
            p.addLine(to: pt(h, side))
            p.closeSubpath()
        }
    }

    // value: a machine with the slash's gap cut off it: the arc of it past
    // the gap's edge, closed along that edge.
    func cutSatellite(_ p: CGMutablePath, _ c: CGPoint) {
        let side: CGFloat = c.x * normal.x + c.y * normal.y > 0 ? 1 : -1
        let away = CGPoint(x: side * normal.x, y: side * normal.y)
        let beyond = side * (c.x * normal.x + c.y * normal.y) - band
        let spread = acos(-beyond / satellite) * 180 / .pi
        let centre = degrees(away)
        p.move(to: polar(centre - spread, satellite, centre: c))
        arc(p, satellite, centre - spread, centre + spread, centre: c, pieces: 3)
        p.closeSubpath()
    }

    func layers(_ state: State) -> (body: CGPath, satellites: CGPath) {
        let body = CGMutablePath(), moons = CGMutablePath()
        let angles: [CGFloat] = state == .working ? [60, 240] : [150, 330]
        let at = angles.map { polar($0, m.ring) }
        let mask = at.map { End.disc($0, clearance) }
        switch state {
        case .plain, .working:
            piece(body, from: mask[1], to: mask[0])
            piece(body, from: mask[0], to: mask[1])
            roundedSquare(body, half: coreHalf, radius: coreRadius)
            for c in at { circle(moons, c, satellite) }
        case .badge:
            let cut = End.disc(dotAt, notch)
            piece(body, from: mask[1], to: cut)
            piece(body, from: cut, to: mask[0])
            piece(body, from: mask[0], to: mask[1])
            roundedSquare(body, half: coreHalf, radius: coreRadius)
            for c in at { circle(moons, c, satellite) }
            circle(moons, dotAt, dot)
        case .slash:
            piece(body, from: mask[1], to: .band(1))
            piece(body, from: mask[0], to: .band(-1))
            slashedCore(body)
            let l = length, r = m.thick / 2
            let a = CGPoint(x: -l * slash.x, y: -l * slash.y), e = CGPoint(x: l * slash.x, y: l * slash.y)
            let along = degrees(slash)
            body.move(to: CGPoint(x: a.x + r * normal.x, y: a.y + r * normal.y))
            body.addLine(to: CGPoint(x: e.x + r * normal.x, y: e.y + r * normal.y))
            arc(body, r, along - 90, along + 90, centre: e, pieces: 2)
            body.addLine(to: CGPoint(x: a.x - r * normal.x, y: a.y - r * normal.y))
            arc(body, r, along + 90, along + 270, centre: a, pieces: 2)
            body.closeSubpath()
            for c in at { cutSatellite(moons, c) }
        }
        return (body, moons)
    }
}

func f(_ v: CGFloat) -> String {
    let r = (v * 1000).rounded() / 1000
    return r == r.rounded() ? String(Int(r)) : String(format: "%.3f", r)
}

func commands(_ path: CGPath) -> String {
    var out = ""
    path.applyWithBlock { out += String(describing: $0.pointee.type.rawValue) }
    return out
}

func svg(_ path: CGPath, dx: CGFloat) -> String {
    let capMiddle = capHeight / 2
    var d = ""
    func p(_ pt: CGPoint) -> String { "\(f(pt.x + dx)) \(f(-(pt.y + capMiddle)))" }
    path.applyWithBlock { e in
        let pts = e.pointee.points
        switch e.pointee.type {
        case .moveToPoint: d += "M\(p(pts[0]))"
        case .addLineToPoint: d += "L\(p(pts[0]))"
        case .addQuadCurveToPoint: d += "Q\(p(pts[0])) \(p(pts[1]))"
        case .addCurveToPoint: d += "C\(p(pts[0])) \(p(pts[1])) \(p(pts[2]))"
        case .closeSubpath: d += "Z"
        @unknown default: break
        }
    }
    return d
}

func template(_ state: State) -> String {
    let line = "style=\"fill:none;stroke:#27AAE1;opacity:1;stroke-width:0.5;\""
    var guides = ""
    var symbols = ""
    for (i, y) in [696, 1126, 1556].enumerated() {
        let tag = ["S", "M", "L"][i]
        guides += "  <line id=\"Baseline-\(tag)\" \(line) x1=\"263\" x2=\"3036\" y1=\"\(y)\" y2=\"\(y)\"/>\n"
        guides += "  <line id=\"Capline-\(tag)\" \(line) x1=\"263\" x2=\"3036\" y1=\"\(f(CGFloat(y) - capHeight))\" y2=\"\(f(CGFloat(y) - capHeight))\"/>\n"
    }
    for m in masters {
        let (body, satellites) = Geometry(m: m).layers(state)
        let shape = commands(body) + "/" + commands(satellites)
        if shape != commands(Geometry(m: masters[0]).layers(state).body) + "/" + commands(Geometry(m: masters[0]).layers(state).satellites) {
            fatalError("\(state.rawValue): \(m.name) is not drawn as \(masters[0].name) is")
        }
        let box = body.boundingBoxOfPath.union(satellites.boundingBoxOfPath)
        let left = m.column
        let dx = m.bearing - box.minX
        let right = left + box.width + 2 * m.bearing
        let top = f(m.baseline - 95), bottom = f(m.baseline + 24)
        guides += "  <line id=\"left-margin-\(m.name)\" \(line) x1=\"\(f(left))\" x2=\"\(f(left))\" y1=\"\(top)\" y2=\"\(bottom)\"/>\n"
        guides += "  <line id=\"right-margin-\(m.name)\" \(line) x1=\"\(f(right))\" x2=\"\(f(right))\" y1=\"\(top)\" y2=\"\(bottom)\"/>\n"
        symbols += "  <g id=\"\(m.name)\" transform=\"matrix(1 0 0 1 \(f(left)) \(f(m.baseline)))\">\n"
        symbols += "   <path class=\"monochrome-0 multicolor-0:tintColor hierarchical-0:secondary SFSymbolsPreviewWireframe\" d=\"\(svg(body, dx: dx))\"/>\n"
        if !satellites.isEmpty {
            symbols += "   <path class=\"monochrome-1 multicolor-1:tintColor hierarchical-1:primary SFSymbolsPreviewWireframe\" d=\"\(svg(satellites, dx: dx))\"/>\n"
        }
        symbols += "  </g>\n"
    }
    return """
    <?xml version="1.0" encoding="UTF-8"?>
    <!DOCTYPE svg PUBLIC "-//W3C//DTD SVG 1.1//EN" "http://www.w3.org/Graphics/SVG/1.1/DTD/svg11.dtd">
    <svg version="1.1" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="3300" height="2200">
     <g id="Notes">
      <text id="template-version" style="stroke:none;fill:black;font-family:sans-serif;font-size:13;" transform="matrix(1 0 0 1 2952 1933)">Template v.3.0</text>
      <text style="stroke:none;fill:black;font-family:sans-serif;font-size:13;" transform="matrix(1 0 0 1 2952 1951)">\(state.rawValue)</text>
     </g>
     <g id="Guides">
    \(guides) </g>
     <g id="Symbols">
    \(symbols) </g>
    </svg>

    """
}

let catalog = URL(fileURLWithPath: CommandLine.arguments[1])
let fm = FileManager.default
try? fm.removeItem(at: catalog)
try fm.createDirectory(at: catalog, withIntermediateDirectories: true)
try #"{ "info": { "author": "xcode", "version": 1 } }"#.appending("\n").write(to: catalog.appendingPathComponent("Contents.json"), atomically: true, encoding: .utf8)
for state in State.allCases {
    let set = catalog.appendingPathComponent("\(state.rawValue).symbolset")
    try fm.createDirectory(at: set, withIntermediateDirectories: true)
    try template(state).write(to: set.appendingPathComponent("\(state.rawValue).svg"), atomically: true, encoding: .utf8)
    let contents = """
    {
        "info": { "author": "xcode", "version": 1 },
        "symbols": [{ "filename": "\(state.rawValue).svg", "idiom": "universal" }]
    }
    """
    try (contents + "\n").write(to: set.appendingPathComponent("Contents.json"), atomically: true, encoding: .utf8)
}

func agentTemplate() -> String {
    let regular = masters.first { $0.name == "Regular-M" }!
    let k = regular.outer / 7.75
    func p(_ x: CGFloat, _ y: CGFloat) -> CGPoint { CGPoint(x: x * k, y: y * k) }
    let lines = CGMutablePath()
    lines.addLines(between: [p(8.4, 5.9), p(13.6, 8.7), p(13.6, 14.6), p(8.4, 17.4), p(3.2, 14.6), p(3.2, 8.7)])
    lines.closeSubpath()
    lines.addLines(between: [p(3.3, 8.7), p(8.4, 11.5), p(13.5, 8.7)])
    lines.addLines(between: [p(8.4, 11.5), p(8.4, 17.3)])
    let cube = lines.copy(strokingWithWidth: regular.stroke, lineCap: .round, lineJoin: .round, miterLimit: 10)
    let sparkle = CGMutablePath()
    sparkle.move(to: p(15.4, 1.2))
    sparkle.addCurve(to: p(18.7, 4.5), control1: p(15.7, 3.1), control2: p(16.8, 4.2))
    sparkle.addCurve(to: p(15.4, 7.8), control1: p(16.8, 4.8), control2: p(15.7, 5.9))
    sparkle.addCurve(to: p(12.1, 4.5), control1: p(15.1, 5.9), control2: p(14.0, 4.8))
    sparkle.addCurve(to: p(15.4, 1.2), control1: p(14.0, 4.2), control2: p(15.1, 3.1))
    sparkle.closeSubpath()
    let mark = cube.union(sparkle)
    let box = mark.boundingBoxOfPath
    var d = ""
    func q(_ pt: CGPoint) -> String { "\(f(pt.x - box.minX + regular.bearing)) \(f(pt.y - box.midY - capHeight / 2))" }
    mark.applyWithBlock { e in
        let pts = e.pointee.points
        switch e.pointee.type {
        case .moveToPoint: d += "M\(q(pts[0]))"
        case .addLineToPoint: d += "L\(q(pts[0]))"
        case .addQuadCurveToPoint: d += "Q\(q(pts[0])) \(q(pts[1]))"
        case .addCurveToPoint: d += "C\(q(pts[0])) \(q(pts[1])) \(q(pts[2]))"
        case .closeSubpath: d += "Z"
        @unknown default: break
        }
    }
    let line = "style=\"fill:none;stroke:#27AAE1;opacity:1;stroke-width:0.5;\""
    let left = regular.column, right = left + box.width + 2 * regular.bearing
    let top = f(regular.baseline - 95), bottom = f(regular.baseline + 24)
    let baselines = sizes.map { size in
        "  <line id=\"Baseline-\(size.name)\" \(line) x1=\"263\" x2=\"3036\" y1=\"\(f(size.baseline))\" y2=\"\(f(size.baseline))\"/>\n"
            + "  <line id=\"Capline-\(size.name)\" \(line) x1=\"263\" x2=\"3036\" y1=\"\(f(size.baseline - capHeight))\" y2=\"\(f(size.baseline - capHeight))\"/>\n"
    }.joined()
    return """
    <?xml version="1.0" encoding="UTF-8"?>
    <!DOCTYPE svg PUBLIC "-//W3C//DTD SVG 1.1//EN" "http://www.w3.org/Graphics/SVG/1.1/DTD/svg11.dtd">
    <svg version="1.1" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="3300" height="2200">
     <g id="Notes">
      <text id="template-version" style="stroke:none;fill:black;font-family:sans-serif;font-size:13;" transform="matrix(1 0 0 1 2952 1933)">Template v.3.0</text>
      <text style="stroke:none;fill:black;font-family:sans-serif;font-size:13;" transform="matrix(1 0 0 1 2952 1951)">station.agent</text>
     </g>
     <g id="Guides">
    \(baselines)      <line id="left-margin-Regular-M" \(line) x1="\(f(left))" x2="\(f(left))" y1="\(top)" y2="\(bottom)"/>
      <line id="right-margin-Regular-M" \(line) x1="\(f(right))" x2="\(f(right))" y1="\(top)" y2="\(bottom)"/>
     </g>
     <g id="Symbols">
      <g id="Regular-M" transform="matrix(1 0 0 1 \(f(left)) \(f(regular.baseline)))">
       <path class="monochrome-0 multicolor-0:tintColor hierarchical-0:primary SFSymbolsPreviewWireframe" d="\(d)"/>
      </g>
     </g>
    </svg>

    """
}

let strokes = "fill=\"none\" stroke=\"#000\" stroke-width=\"1.6\" stroke-linecap=\"round\" stroke-linejoin=\"round\""
let marks: [String: String] = [
    "claude": "<g stroke=\"#000\" stroke-width=\"1.9\" stroke-linecap=\"round\">" + [0, 30, 60, 90, 120, 150].map { "<line x1=\"10\" y1=\"2.6\" x2=\"10\" y2=\"17.4\" transform=\"rotate(\($0) 10 10)\"/>" }.joined() + "</g>",
    "npm": "<path fill=\"#000\" fill-rule=\"evenodd\" d=\"M1.5 6.5h17v6.4h-8.5v1.3H6.2v-1.3H1.5zM3 8v3.4h1.9V9.3h1v2.1h1V8zm4.9 0v4.6h1.9v-1.2h1.9V8zm1.9 1.2h.9v1h-.9zM12.6 8v3.4h1.9V9.3h.9v2.1h1V9.3h.9v2.1h1V8z\"/>",
    "tailscale": [(4, 4, 0.35), (10, 4, 0.35), (16, 4, 0.35), (4, 10, 1), (10, 10, 1), (16, 10, 1), (4, 16, 0.35), (10, 16, 1), (16, 16, 0.35)].map { "<circle cx=\"\($0.0)\" cy=\"\($0.1)\" r=\"2.1\" fill=\"#000\" opacity=\"\($0.2)\"/>" }.joined(),
    "cloudflare": "<path d=\"M5 15h10.4a3 3 0 0 0 .3-6 4.6 4.6 0 0 0-8.8-.8A3.5 3.5 0 0 0 5 15z\" fill=\"#000\"/>",
    "docker": "<g fill=\"#000\"><rect x=\"3\" y=\"9\" width=\"2.6\" height=\"2.4\"/><rect x=\"6\" y=\"9\" width=\"2.6\" height=\"2.4\"/><rect x=\"9\" y=\"9\" width=\"2.6\" height=\"2.4\"/><rect x=\"6\" y=\"6.2\" width=\"2.6\" height=\"2.4\"/><rect x=\"9\" y=\"6.2\" width=\"2.6\" height=\"2.4\"/><rect x=\"12\" y=\"9\" width=\"2.6\" height=\"2.4\"/><path d=\"M2 12h15c.8-.4 1.2-1.2 1.4-2 .2 1.6-.6 3-2 3.6-1.6 2.6-4.6 3.4-7.8 3.4-3.4 0-5.8-1.6-6.6-5z\"/></g>",
    "github": "<g transform=\"scale(1.25)\"><path fill=\"#000\" d=\"M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.013 8.013 0 0016 8c0-4.42-3.58-8-8-8z\"/></g>",
    "vite": "<path d=\"M11 2.5L4.8 11h4.6l-.9 6.5L15.2 9h-4.6z\" fill=\"#000\"/>",
    "astro": "<path d=\"M10 2.8l4.6 11.4c-1.3-.7-2.8-1-4.6-1s-3.3.3-4.6 1z\" fill=\"#000\"/><path \(strokes) d=\"M8.4 15.4c.2 1 .8 1.8 1.6 1.8s1.4-.8 1.6-1.8\"/>",
    "postgres": "<g \(strokes)><ellipse cx=\"10\" cy=\"5.4\" rx=\"5\" ry=\"2\"/><path d=\"M5 5.4v9.2c0 1.1 2.2 2 5 2s5-.9 5-2V5.4M5 10c0 1.1 2.2 2 5 2s5-.9 5-2\"/></g>",
    "traefik": "<g \(strokes)><circle cx=\"10\" cy=\"10\" r=\"6.6\"/><path d=\"M3.4 10h13.2M10 3.4c1.8 1.8 2.7 4 2.7 6.6s-.9 4.8-2.7 6.6c-1.8-1.8-2.7-4-2.7-6.6s.9-4.8 2.7-6.6z\"/></g>",
    "kubernetes": "<g \(strokes)><circle cx=\"10\" cy=\"10\" r=\"2.2\"/><path d=\"M10 3v4.8M10 12.2V17M3.3 7.8l4.6 1.5M12.1 10.7l4.6 1.5M5.6 15.6l2.8-3.8M11.6 8.2l2.8-3.8M14.4 15.6l-2.8-3.8M8.4 8.2L5.6 4.4\"/></g>",
    "node": "<g \(strokes)><path d=\"M10 2.8l6.2 3.6v7.2L10 17.2l-6.2-3.6V6.4z\"/><path d=\"M10 7.2v5.6\"/></g>",
    "storybook": "<path \(strokes) d=\"M13.4 6.4c-.6-.9-1.8-1.4-3.2-1.4-1.8 0-3 .9-3 2.3 0 3.2 6.4 1.8 6.4 5.1 0 1.5-1.4 2.6-3.4 2.6-1.6 0-2.9-.6-3.6-1.7\"/>",
    "api": "<path \(strokes) d=\"M7.5 4.5c-1.6 0-2.2.8-2.2 2.2v1.6c0 .9-.5 1.5-1.5 1.7 1 .2 1.5.8 1.5 1.7v1.6c0 1.4.6 2.2 2.2 2.2M12.5 4.5c1.6 0 2.2.8 2.2 2.2v1.6c0 .9.5 1.5 1.5 1.7-1 .2-1.5.8-1.5 1.7v1.6c0 1.4-.6 2.2-2.2 2.2\"/>",
    "web": "<g \(strokes)><rect x=\"3\" y=\"4\" width=\"14\" height=\"12\" rx=\"2.4\"/><path d=\"M3 7.6h14\"/></g><circle cx=\"5.4\" cy=\"5.8\" r=\".5\" fill=\"#000\"/><circle cx=\"7.2\" cy=\"5.8\" r=\".5\" fill=\"#000\"/>",
]

let agent = catalog.appendingPathComponent("station.agent.symbolset")
try fm.createDirectory(at: agent, withIntermediateDirectories: true)
try agentTemplate().write(to: agent.appendingPathComponent("station.agent.svg"), atomically: true, encoding: .utf8)
try """
{
    "info": { "author": "xcode", "version": 1 },
    "symbols": [{ "filename": "station.agent.svg", "idiom": "universal" }]
}

""".write(to: agent.appendingPathComponent("Contents.json"), atomically: true, encoding: .utf8)

for (service, body) in marks.sorted(by: { $0.key < $1.key }) {
    let name = "station.mark.\(service)"
    let set = catalog.appendingPathComponent("\(name).imageset")
    try fm.createDirectory(at: set, withIntermediateDirectories: true)
    let svg = "<svg xmlns=\"http://www.w3.org/2000/svg\" width=\"20\" height=\"20\" viewBox=\"0 0 20 20\">\(body)</svg>\n"
    try svg.write(to: set.appendingPathComponent("\(name).svg"), atomically: true, encoding: .utf8)
    try """
    {
        "images": [{ "filename": "\(name).svg", "idiom": "universal" }],
        "info": { "author": "xcode", "version": 1 },
        "properties": { "preserves-vector-representation": true, "template-rendering-intent": "template" }
    }

    """.write(to: set.appendingPathComponent("Contents.json"), atomically: true, encoding: .utf8)
}
