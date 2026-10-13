import SwiftUI
import UIKit

enum GlassTheme {
    static let accent = Color(uiColor: .tuiAccent)
    static let background = Color(uiColor: .tuiBackground)
    static let glow = Color(uiColor: .tuiGlow)
    static let text = Color(uiColor: .tuiText)
    static let muted = Color(uiColor: .tuiMuted)
}

struct GlassBackground: View {
    var body: some View {
        ZStack {
            LinearGradient(colors: [GlassTheme.background, GlassTheme.glow.opacity(0.7), GlassTheme.background], startPoint: .topLeading, endPoint: .bottomTrailing)
                .ignoresSafeArea()
            // Ambient soft aurora glow in upper-left
            Circle()
                .fill(GlassTheme.accent.opacity(0.08))
                .frame(width: 280, height: 280)
                .blur(radius: 60)
                .offset(x: -80, y: -180)
        }
    }
}

struct BrandHeader: View {
    let title: String
    let subtitle: String
    var body: some View {
        HStack(spacing: 16) {
            Image("TuiGoLogo").resizable().scaledToFill()
                .frame(width: 76, height: 76)
                .clipShape(RoundedRectangle(cornerRadius: 20, style: .continuous))
                .overlay(RoundedRectangle(cornerRadius: 20, style: .continuous).stroke(Color.white.opacity(0.4), lineWidth: 1.5))
                .shadow(color: GlassTheme.accent.opacity(0.25), radius: 10, y: 4)
                .accessibilityLabel("Logo Tui Gõ")
            VStack(alignment: .leading, spacing: 6) {
                Text(title).font(.largeTitle.bold()).lineLimit(1).minimumScaleFactor(0.8)
                Text(subtitle).font(.subheadline).foregroundColor(GlassTheme.muted)
            }
            Spacer(minLength: 0)
        }
    }
}

struct GlassCard<Content: View>: View {
    let title: String
    let subtitle: String
    let content: Content
    @Environment(\.accessibilityReduceTransparency) private var reduceTransparency
    init(_ title: String, subtitle: String, @ViewBuilder content: () -> Content) {
        self.title = title; self.subtitle = subtitle; self.content = content()
    }
    var body: some View {
        VStack(alignment: .leading, spacing: 14) {
            VStack(alignment: .leading, spacing: 5) {
                Text(title).font(.title3.weight(.semibold))
                Text(subtitle).font(.footnote).foregroundColor(GlassTheme.muted)
            }
            content
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(20)
        .background {
            if reduceTransparency {
                RoundedRectangle(cornerRadius: 24, style: .continuous).fill(Color(uiColor: .tuiSurface))
            } else {
                RoundedRectangle(cornerRadius: 24, style: .continuous)
                    .fill(.ultraThinMaterial)
                    .overlay(
                        LinearGradient(
                            colors: [Color.white.opacity(0.12), Color.white.opacity(0.02)],
                            startPoint: .top,
                            endPoint: .bottom
                        )
                        .clipShape(RoundedRectangle(cornerRadius: 24, style: .continuous))
                    )
            }
        }
        .overlay(
            RoundedRectangle(cornerRadius: 24, style: .continuous)
                .stroke(
                    LinearGradient(
                        colors: [Color.white.opacity(0.35), Color(uiColor: .tuiBorder).opacity(0.6)],
                        startPoint: .top,
                        endPoint: .bottom
                    ),
                    lineWidth: 1
                )
        )
        .shadow(color: .black.opacity(0.05), radius: 14, y: 6)
    }
}

extension UIColor {
    private static func adaptive(_ light: UInt32, _ dark: UInt32) -> UIColor {
        UIColor { traits in
            let rgb = traits.userInterfaceStyle == .dark ? dark : light
            return UIColor(red: CGFloat((rgb >> 16) & 255) / 255, green: CGFloat((rgb >> 8) & 255) / 255, blue: CGFloat(rgb & 255) / 255, alpha: 1)
        }
    }
    static let tuiBackground = adaptive(0xF8FAFC, 0x0F1115)
    static let tuiGlow = adaptive(0xE2F0E7, 0x243D3B)
    static let tuiSurface = adaptive(0xFFFFFF, 0x181A20)
    static let tuiRaised = adaptive(0xF1F5F9, 0x222630)
    static let tuiBorder = adaptive(0xE2E8F0, 0x272B35)
    static let tuiText = adaptive(0x0F172A, 0xF8FAFC)
    static let tuiMuted = adaptive(0x64748B, 0x94A3B8)
    static let tuiAccent = adaptive(0x197F64, 0x3ED3B8)
    static let tuiPressed = adaptive(0x146B54, 0x65DFC9)
    static let tuiOnAccent = adaptive(0xFFFFFF, 0x0F1115)
}
