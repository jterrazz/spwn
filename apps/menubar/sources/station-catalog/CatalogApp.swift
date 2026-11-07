import AppKit
import StationKit
import SwiftUI

struct CatalogApp: App {
    @StateObject private var catalog = Catalog()

    init() {
        NSApplication.shared.setActivationPolicy(.regular)
    }

    var body: some Scene {
        WindowGroup("Station Catalog") {
            CatalogWindow()
                .environmentObject(catalog)
                .frame(minWidth: 900, minHeight: 600)
        }
        .defaultSize(width: 1100, height: 820)
    }
}

struct CatalogWindow: View {
    @EnvironmentObject private var catalog: Catalog
    @State private var inspector = true

    var body: some View {
        NavigationSplitView {
            Sidebar()
                .navigationSplitViewColumnWidth(min: 240, ideal: 280)
        } detail: {
            Canvas()
                .inspector(isPresented: $inspector) {
                    Inspector().inspectorColumnWidth(280)
                }
        }
        .searchable(text: $catalog.search, placement: .sidebar, prompt: "Filter ids and notes")
        .toolbar { Knobs(inspector: $inspector) }
    }
}

struct Sidebar: View {
    @EnvironmentObject private var catalog: Catalog

    var body: some View {
        if let failure = catalog.failure {
            Text(failure).foregroundStyle(.secondary).padding()
        } else {
            List(selection: $catalog.selection) {
                ForEach(catalog.shown, id: \.group) { section in
                    Section(section.group.rawValue) {
                        ForEach(section.stories) { story in
                            Row(story: story, drifted: catalog.drifted.contains(story.id)).tag(story.id)
                        }
                    }
                }
            }
        }
    }

    struct Row: View {
        let story: Story
        let drifted: Bool

        var body: some View {
            HStack(alignment: .firstTextBaseline, spacing: 6) {
                VStack(alignment: .leading, spacing: 2) {
                    Text(story.id).font(.body.monospaced())
                    Text(story.note).font(.callout).foregroundStyle(.secondary).lineLimit(2)
                }
                Spacer(minLength: 0)
                if drifted {
                    Image(systemName: "circle.fill").font(.system(size: 7)).foregroundStyle(.red)
                        .help("Its golden differs from its last run")
                }
            }
        }
    }
}

struct Canvas: View {
    @EnvironmentObject private var catalog: Catalog

    var body: some View {
        ScrollView([.horizontal, .vertical]) {
            if catalog.trial {
                TrialPanel { line in FileHandle.standardError.write(Data("trial \(line)\n".utf8)) }
                    .stationAccessibility(reduceMotion: catalog.knobs.reduceMotion)
                    .padding(40)
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else if let story = catalog.current {
                VStack(spacing: 8) {
                    stage(story)
                    Ruler(width: story.ui.statusItem == true ? StoryStage.strip.width : catalog.knobs.width)
                }
                .padding(40)
                .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else {
                Text("No story").foregroundStyle(.secondary).padding(40)
            }
        }
        .background(StoryDeskView(catalog.knobs.desk, scheme: catalog.knobs.scheme))
    }

    @ViewBuilder
    func stage(_ story: Story) -> some View {
        if catalog.live, let session = catalog.session {
            LiveStage(story: story, knobs: catalog.knobs, session: session)
        } else if let image = Self.still(story, knobs: catalog.knobs) {
            Image(nsImage: image).interpolation(.none)
        }
    }

    static func still(_ story: Story, knobs: StoryKnobs) -> NSImage? {
        guard let png = StoryStage.png(story, knobs: knobs), let rep = NSBitmapImageRep(data: png) else { return nil }
        let image = NSImage(size: NSSize(width: CGFloat(rep.pixelsWide) / knobs.scale, height: CGFloat(rep.pixelsHigh) / knobs.scale))
        image.addRepresentation(rep)
        return image
    }

    struct LiveStage: View {
        let story: Story
        let knobs: StoryKnobs
        @ObservedObject var session: Live

        var body: some View {
            StoryStage(story, knobs: knobs, state: session.state, handlers: session.handlers)
        }
    }

    struct Ruler: View {
        let width: CGFloat

        var body: some View {
            VStack(spacing: 2) {
                Rectangle().frame(width: width, height: 1)
                Text("\(Int(width)) pt").font(.caption2.monospacedDigit())
            }
            .foregroundStyle(.secondary)
        }
    }
}

struct Inspector: View {
    @EnvironmentObject private var catalog: Catalog

    var body: some View {
        ScrollView {
            if let story = catalog.current {
                VStack(alignment: .leading, spacing: 14) {
                    Text(story.id).font(.headline.monospaced())
                    Text(story.note)
                    Field(title: "Data", text: story.file.pages.joined(separator: "\n"))
                    if let ui = Self.json(story.file.ui) { Field(title: "Interactions", text: ui) }
                    if let play = story.play { Field(title: "Play", text: "from \(play.from), \(play.gesture)") }
                    if let note = story.file.heightNote { Field(title: "Height", text: note) }
                    Capture(id: story.id, dark: catalog.knobs.scheme == .dark)
                    if catalog.live, let session = catalog.session { Calls(session: session) }
                }
                .frame(maxWidth: .infinity, alignment: .leading)
                .padding()
            }
        }
    }

    static func json<T: Encodable>(_ value: T?) -> String? {
        guard let value else { return nil }
        let encoder = JSONEncoder()
        encoder.outputFormatting = [.sortedKeys, .prettyPrinted, .withoutEscapingSlashes]
        return (try? encoder.encode(value)).flatMap { String(data: $0, encoding: .utf8) }
    }

    struct Field: View {
        let title: String
        let text: String

        var body: some View {
            VStack(alignment: .leading, spacing: 4) {
                Text(title).font(.caption).foregroundStyle(.secondary)
                Text(text).font(.callout.monospaced()).textSelection(.enabled)
            }
        }
    }

    struct Capture: View {
        let id: String
        let dark: Bool

        var body: some View {
            VStack(alignment: .leading, spacing: 4) {
                Text("Prototype").font(.caption).foregroundStyle(.secondary)
                if let url = Paths.capture(id, dark: dark), let image = NSImage(contentsOf: url) {
                    Image(nsImage: image).resizable().scaledToFit()
                } else {
                    Text(Paths.design == nil ? "Set STATION_DESIGN to the design folder" : "No capture of \(id)")
                        .font(.callout).foregroundStyle(.secondary)
                }
            }
        }
    }

    struct Calls: View {
        @ObservedObject var session: Live

        var body: some View {
            Field(title: "Calls", text: session.calls.isEmpty ? "none yet" : session.calls.joined(separator: "\n"))
        }
    }
}

struct Knobs: ToolbarContent {
    @EnvironmentObject private var catalog: Catalog
    @Binding var inspector: Bool

    var body: some ToolbarContent {
        ToolbarItemGroup {
            Picker("Appearance", selection: $catalog.knobs.scheme) {
                Text("Dark").tag(ColorScheme.dark)
                Text("Light").tag(ColorScheme.light)
            }
            .pickerStyle(.segmented)
            Toggle("Increase Contrast", isOn: $catalog.knobs.increaseContrast)
            Toggle("Reduce Transparency", isOn: $catalog.knobs.reduceTransparency)
            Toggle("Reduce Motion", isOn: $catalog.knobs.reduceMotion)
            Picker("Width", selection: $catalog.knobs.width) {
                ForEach(StoryKnobs.widths, id: \.self) { Text("\(Int($0))").tag($0) }
            }
            Picker("Scale", selection: $catalog.knobs.scale) {
                ForEach(StoryKnobs.scales, id: \.self) { Text("\(Int($0))×").tag($0) }
            }
            Picker("Desk", selection: $catalog.knobs.desk) {
                ForEach(StoryDesk.allCases, id: \.self) { Text($0.rawValue.capitalized).tag($0) }
            }
            Picker("Accent", selection: $catalog.knobs.accent) {
                ForEach(StoryAccent.allCases, id: \.self) { Text($0 == .blue ? "Blue" : "System").tag($0) }
            }
        }
        ToolbarItemGroup {
            Toggle("Live", isOn: $catalog.live)
            Button("Play", systemImage: "play.fill") { catalog.play() }.disabled(!catalog.playable)
            Picker("Speed", selection: $catalog.speed) {
                ForEach(Catalog.speeds, id: \.self) { Text("\($0, specifier: "%g")×").tag($0) }
            }
            Slider(value: $catalog.progress, in: 0...1) { Text("Freeze") }
                .frame(width: 120)
                .disabled(catalog.current?.freeze == nil)
            Button("Reload", systemImage: "arrow.clockwise") { catalog.reload() }
            Toggle("Inspector", systemImage: "sidebar.trailing", isOn: $inspector)
        }
    }
}
