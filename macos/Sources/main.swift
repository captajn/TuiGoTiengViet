import Cocoa
import InputMethodKit
import SwiftUI

// Tui Go input method for macOS — InputMethodKit server + settings window
// in one process. The OS launches this bundle from ~/Library/Input Methods.

let kConnectionName = "TuiGo_Connection"

final class AppDelegate: NSObject, NSApplicationDelegate {
    static private(set) var shared: AppDelegate?
    private var settingsWindow: NSWindow?

    override init() {
        super.init()
        AppDelegate.shared = self
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
_ = IMKServer(name: kConnectionName,
              controllerClass: InputController.self,
              delegateClass: nil)
NSApplication.shared.run()
