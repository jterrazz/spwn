import AppKit

// invariant: each picture of the iconset is its own size's drawing rendered at exactly its pixel
// count, never a larger one scaled down, so the hand-drawn 16 and 32 px sizes are the ones shipped.
let usage = "usage: swift app-icon-iconset.swift <sizes dir of spwn-icon-<px>.svg> <out .iconset>"
let arguments = CommandLine.arguments
guard arguments.count == 3 else {
    FileHandle.standardError.write(Data((usage + "\n").utf8))
    exit(2)
}
let sizes = URL(fileURLWithPath: arguments[1]), iconset = URL(fileURLWithPath: arguments[2])
// value: an iconset picture's file name, to the pixel size of the drawing it holds.
let pictures: [(String, Int)] = [
    ("icon_16x16.png", 16), ("icon_16x16@2x.png", 32), ("icon_32x32.png", 32), ("icon_32x32@2x.png", 64),
    ("icon_128x128.png", 128), ("icon_128x128@2x.png", 256), ("icon_256x256.png", 256), ("icon_256x256@2x.png", 512),
    ("icon_512x512.png", 512), ("icon_512x512@2x.png", 1024),
]
try FileManager.default.createDirectory(at: iconset, withIntermediateDirectories: true)
for (name, pixels) in pictures {
    let source = sizes.appendingPathComponent("spwn-icon-\(pixels).svg")
    guard let drawing = NSImage(contentsOf: source),
          let rep = NSBitmapImageRep(bitmapDataPlanes: nil, pixelsWide: pixels, pixelsHigh: pixels, bitsPerSample: 8, samplesPerPixel: 4,
                                     hasAlpha: true, isPlanar: false, colorSpaceName: .deviceRGB, bytesPerRow: 0, bitsPerPixel: 0)
    else {
        FileHandle.standardError.write(Data("app-icon-iconset: cannot draw \(source.path)\n".utf8))
        exit(1)
    }
    rep.size = NSSize(width: pixels, height: pixels)
    NSGraphicsContext.saveGraphicsState()
    NSGraphicsContext.current = NSGraphicsContext(bitmapImageRep: rep)
    drawing.draw(in: NSRect(x: 0, y: 0, width: pixels, height: pixels), from: .zero, operation: .copy, fraction: 1)
    NSGraphicsContext.restoreGraphicsState()
    let srgb = rep.converting(to: .sRGB, renderingIntent: .default) ?? rep
    guard let png = srgb.representation(using: .png, properties: [:]) else { exit(1) }
    try png.write(to: iconset.appendingPathComponent(name))
}
print("app-icon-iconset: \(pictures.count) pictures into \(iconset.path)")
