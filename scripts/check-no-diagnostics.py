"""Reject app-owned diagnostics in shipped sources on all five platforms.

Build/test scripts and legacy C++ references are not shipped app loggers.
OS/framework crash reporting is outside this source-level policy.
"""
from pathlib import Path
import argparse
import re
import sys
import plistlib
import xml.etree.ElementTree as ET

PRODUCTS = {
    "Windows": ("app", ".go"),
    "Linux": ("linux", ".go"),
    "Android app": ("android/app/src/main", ".kt"),
    "Android engine": ("android/engine/src/main", ".kt"),
    "iOS": ("ios", ".swift"),
    "macOS": ("macos/Sources", ".swift"),
}
FORBIDDEN = [
    ("diagnostic file", r'["\'][^"\'\n]*\.(?:log|dmp)["\']'),
    ("diagnostic helper", r'\b(?:logLine|recoverCrash|wlog)\s*\('),
    ("Go logger", r'\b(?:log|slog)\.(?:Print\w*|Fatal\w*|Panic\w*|Debug\w*|Info\w*|Warn\w*|Error\w*|Log\w*|SetOutput|New)\s*\('),
    ("console diagnostics", r'\b(?:fmt\.Print\w*|print|println|NSLog|os_log|os_signpost|debugPrint|dump)\s*\('),
    ("diagnostic stream", r'\bos\.(?:Stderr|Stdout)\b|\bFileHandle\.standard(?:Error|Output)\b'),
    ("crash redirection", r'\b(?:SetCrashOutput|SetTraceback|SetStdHandle|STD_ERROR_HANDLE)\b'),
    ("Android logger", r'\b(?:android\.util\.Log|Log\.(?:v|d|i|w|e|wtf)|Timber\.\w+)\b'),
    ("Apple logger", r'\b(?:Logger|OSLog)\s*\('),
    ("logging option", r'\b(?:DebugLog|DebugLogging|KEY_DEBUG_LOG|keyDebugLog)\b'),
    ("telemetry SDK", r'\b(?:FirebaseAnalytics|Crashlytics|Sentry|AppCenter|Amplitude|Mixpanel)\b'),
]
NETWORK = r'net/http|\bnet\.Dial\w*\s*\(|\b(?:URLSession|URLRequest|HttpURLConnection|OkHttpClient|WebSocket)\b|\bretrofit2\b'


def check(root):
    failures = []
    for platform, (directory, suffix) in PRODUCTS.items():
        base = root / directory
        files = sorted(p for p in base.rglob("*" + suffix)
                       if not p.name.endswith("_test.go") and
                       not any(part in {"Tests", ".build", "build", "dist", "test"}
                               for part in p.relative_to(base).parts))
        if not files:
            failures.append(f"{platform}: no product files found under {directory}")
        for path in files:
            for number, line in enumerate(path.read_text(encoding="utf-8-sig").splitlines(), 1):
                # Ignore comment-only lines (e.g. explaining the no-logging policy).
                if line.lstrip().startswith(("//", "*", "/*")):
                    continue
                for reason, expression in FORBIDDEN:
                    if re.search(expression, line):
                        failures.append(f"{path.relative_to(root)}:{number}: {reason}")
                # Only the separate, optional Windows updater may use HTTP.
                if not path.relative_to(root).as_posix().startswith("app/updater/") and re.search(NETWORK, line):
                    failures.append(f"{path.relative_to(root)}:{number}: network client in input app")
        print(f"{platform}: checked {len(files)} product files")
    manifest = ET.parse(root / "android/app/src/main/AndroidManifest.xml").getroot()
    permissions = [node.get("{http://schemas.android.com/apk/res/android}name")
                   for node in manifest.findall("uses-permission")]
    if "android.permission.INTERNET" in permissions:
        failures.append("Android manifest requests INTERNET")
    application = manifest.find("application")
    if application is None or application.get("{http://schemas.android.com/apk/res/android}allowBackup") != "false":
        failures.append("Android app backup must remain disabled")
    with (root / "ios/Keyboard/Info.plist").open("rb") as stream:
        info = plistlib.load(stream)
    if info["NSExtension"]["NSExtensionAttributes"].get("RequestsOpenAccess") is not False:
        failures.append("iOS keyboard requests Full Access")
    print("Privacy: no input-app HTTP client/telemetry SDK; Android no INTERNET/backup; iOS no Full Access request")
    return failures


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parents[1])
    args = parser.parse_args()
    failures = check(args.root)
    if failures:
        print("FAIL: app-owned diagnostics found", file=sys.stderr)
        print("\n".join(failures), file=sys.stderr)
        sys.exit(1)
    print("PASS: no app-owned diagnostic logging or debug-log option on all five platforms")
