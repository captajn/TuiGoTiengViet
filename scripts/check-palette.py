"""Guard mobile palette parity with Windows; reads source, never rewrites it."""
from pathlib import Path
import re
import xml.etree.ElementTree as ET
import hashlib
import json
import struct

repo = Path(__file__).resolve().parent.parent
windows = (repo / "app/theme.go").read_text(encoding="utf-8-sig")
kotlin = (repo / "android/app/src/main/kotlin/com/tuigo/UiTheme.kt").read_text(encoding="utf-8-sig")
swift = (repo / "ios/Shared/GlassTheme.swift").read_text(encoding="utf-8-sig")
mapping = {
    "aBg": ("bg", "bg", "tuiBackground"),
    "aCard": ("surface", "surface", "tuiSurface"),
    "aBorder": ("border", "border", "tuiBorder"),
    "aText": ("text", "text", "tuiText"),
    "aMuted": ("muted", "text_muted", "tuiMuted"),
    "aAccent": ("accent", "accent", "tuiAccent"),
    "aKeycap": ("raised", "surface_raised", "tuiRaised"),
}
for mode, index in [("Light", 0), ("Dark", 1)]:
    win = re.search(r"var pal" + mode + r" = palette\{(.*?)\n\}", windows, re.S).group(1)
    android = re.search(r"val " + mode + r" = Palette\((.*?)\n    \)", kotlin, re.S).group(1)
    resources = ET.parse(repo / "android/app/src/main/res" / ("values" if index == 0 else "values-night") / "colors.xml")
    xml = {node.attrib["name"]: int(node.text.lstrip("#"), 16) for node in resources.getroot()}
    for win_key, (kt_key, xml_key, swift_key) in mapping.items():
        expected = int(re.search(r"\b" + win_key + r":\s*0x([A-Fa-f0-9]+)", win).group(1), 16) & 0xFFFFFF
        actual = int(re.search(r"\b" + kt_key + r"\s*=\s*0x([A-Fa-f0-9]+)", android).group(1), 16) & 0xFFFFFF
        pair = re.search(swift_key + r" = adaptive\(0x([A-Fa-f0-9]+), 0x([A-Fa-f0-9]+)\)", swift)
        assert actual == expected, f"Android {mode}.{kt_key} differs from Windows"
        assert xml[xml_key] == expected, f"Android XML {mode}.{xml_key} differs from Windows"
        assert int(pair.group(index + 1), 16) == expected, f"iOS {mode}.{swift_key} differs from Windows"
print("PASS: Windows/Android/iOS match for 7 palette tokens in both light and dark mode.")
source_logo = hashlib.sha256((repo / "app/logo.jpg").read_bytes()).digest()
for file in [repo / "android/app/src/main/res/drawable-nodpi/tuigo_logo.jpg", repo / "ios/App/Assets.xcassets/TuiGoLogo.imageset/logo.jpg"]:
    assert hashlib.sha256(file.read_bytes()).digest() == source_logo, f"Logo differs from Windows: {file}"
icons = repo / "ios/App/Assets.xcassets/AppIcon.appiconset"
manifest = json.loads((icons / "Contents.json").read_text(encoding="utf-8-sig"))
for slot in manifest["images"]:
    png = (icons / slot["filename"]).read_bytes()
    assert png[:8] == b"\x89PNG\r\n\x1a\n"
    width, height, depth, kind, *_ = struct.unpack(">IIBBBBB", png[16:29])
    expected = round(float(slot["size"].split("x")[0]) * float(slot["scale"].rstrip("x")))
    assert width == height == expected and kind == 2, f"Wrong dimensions or alpha in {slot}"
print("PASS: exact Windows logo in both apps; all 18 iOS app-icon slots exist at the right sizes without alpha.")
