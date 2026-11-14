import Cocoa
import InputMethodKit
import SwiftUI

// Tui Go input method for macOS — InputMethodKit server + settings window
// in one process. The OS launches this bundle from ~/Library/Input Methods.

let kConnectionName = "TuiGo_Connection"

final class AppDelegate: NSObject, NSApplicationDelegate {
    static private(set) var shared: AppDelegate?
    private var settingsWindow: NSWindow?
    // The server must stay referenced for the lifetime of the process; a bare
    // `_ = IMKServer(...)` lets ARC release it right after the call.
    private var server: IMKServer?

    override init() {
        super.init()
        AppDelegate.shared = self
    }

    // IMKServer(name:bundleIdentifier:) reads InputMethodServerControllerClass
    // from the bundle. The older name:controllerClass:delegateClass: variant
    // faults on macOS 15: in _IMKServerLegacy it writes through _private before
    // that ivar is allocated, so it crashes at address 0x8 regardless of how
    // the bundle was installed.
    func startServer(connectionName: String, bundleIdentifier: String) {
        server = IMKServer(name: connectionName, bundleIdentifier: bundleIdentifier)
    }

    static func showSettings() {
        shared?.showSettingsWindow()
    }

    private func showSettingsWindow() {
        if settingsWindow == nil {
            let view = SettingsView()
            let hosting = NSHostingController(rootView: view)
            let win = NSWindow(contentViewController: hosting)
            win.title = "Cài đặt Tui Gõ"
            win.styleMask = [.titled, .closable, .miniaturizable]
            win.setContentSize(NSSize(width: 520, height: 620))
            win.isReleasedWhenClosed = false
            settingsWindow = win
        }
        NSApp.setActivationPolicy(.accessory)
        NSApp.activate(ignoringOtherApps: true)
        settingsWindow?.makeKeyAndOrderFront(nil)
    }

    func applicationShouldTerminateAfterLastWindowClosed(_ sender: NSApplication) -> Bool {
        false // the IMK server must keep running
    }
}

let delegate = AppDelegate()
NSApplication.shared.delegate = delegate
if let bundleIdentifier = Bundle.main.bundleIdentifier {
    delegate.startServer(connectionName: kConnectionName, bundleIdentifier: bundleIdentifier)
}
NSApplication.shared.run()
