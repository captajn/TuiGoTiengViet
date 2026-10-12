// swift-tools-version: 5.9
import PackageDescription

let package = Package(
    name: "TuiGoEngine",
    platforms: [
        .iOS(.v15),
        .macOS(.v12),
    ],
    products: [
        .library(name: "TuiGoEngine", targets: ["TuiGoEngine"]),
    ],
    targets: [
        .target(name: "TuiGoEngine"),
        .testTarget(
            name: "TuiGoEngineTests",
            dependencies: ["TuiGoEngine"]
        ),
    ]
)
