// swift-tools-version: 6.2
import PackageDescription

let package = Package(
    name: "StationMenuBar",
    platforms: [.macOS(.v26)],
    products: [
        .library(name: "SpwnMenuBar", type: .dynamic, targets: ["SpwnMenuBar"]),
        .executable(name: "StationMenuBar", targets: ["StationMenuBar"]),
        .executable(name: "StationCatalog", targets: ["StationCatalog"]),
    ],
    targets: [
        .target(name: "StationKit", path: "sources/station-kit", resources: [.process("token/symbols.xcassets")]),
        .target(name: "SpwnMenuBar", dependencies: ["StationKit"], path: "sources/spwn-menu-bar"),
        .executableTarget(name: "StationMenuBar", dependencies: ["SpwnMenuBar"], path: "sources/station-menu-bar"),
        .executableTarget(name: "StationCatalog", dependencies: ["StationKit"], path: "sources/station-catalog"),
        .testTarget(name: "StationKitTests", dependencies: ["StationKit"], path: "tests/station-kit-tests", exclude: ["__expected__"]),
        .testTarget(name: "StationCatalogTests", dependencies: ["StationCatalog", "StationKit"], path: "tests/station-catalog-tests"),
    ],
    swiftLanguageModes: [.v5]
)
