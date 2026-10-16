#!/usr/bin/env python3
"""Shared mobile version metadata and deterministic artifact packaging (stdlib only)."""
import argparse
import json
import os
from pathlib import Path
import plistlib
import re
import shutil
import zipfile

ROOT = Path(__file__).resolve().parents[1]


def metadata(tag=None):
    android_text = (ROOT / 'android/app/build.gradle.kts').read_text()
    android_match = re.search(r'versionName\s*=.*?orElse\("([^"]+)"\)', android_text)
    android = android_match.group(1) if android_match else re.search(r'versionName\s*=\s*"([^"]+)"', android_text).group(1)
    ios = re.search(r'MARKETING_VERSION:\s*"([^"]+)"', (ROOT / 'ios/project.yml').read_text()).group(1)
    if not re.fullmatch(r'\d+\.\d+\.\d+', android) or ios != '$(TUIGO_VERSION)':
        raise ValueError('Android/iOS versions must match and use major.minor.patch')
    version = tag.removeprefix('mobile-v') if tag else '0.1.5'
    if not re.fullmatch(r'\d+\.\d+\.\d+', version):
        raise ValueError(f'Mobile version must use major.minor.patch (got {version!r})')
    return dict(version=version, apk_name=f'tuigo-v{version}.apk', ipa_name=f'tuigo-v{version}.ipa', release_name=f'Tui Gõ Mobile v{version}')


def check_ipa(path, version):
    with zipfile.ZipFile(path) as archive:
        names = archive.namelist()
        apps = [n for n in names if re.fullmatch(r'Payload/[^/]+\.app/Info.plist', n)]
        if len(apps) != 1:
            raise ValueError('IPA must contain exactly one Payload app')
        base = apps[0].removesuffix('Info.plist')
        app = plistlib.loads(archive.read(apps[0]))
        if app['CFBundleShortVersionString'] != version or app['CFBundleIdentifier'] != 'com.tuigo':
            raise ValueError('IPA app version or bundle ID mismatch')
        extensions = [n for n in names if re.fullmatch(re.escape(base) + r'PlugIns/[^/]+\.appex/Info.plist', n)]
        if len(extensions) != 1:
            raise ValueError('IPA must embed exactly one keyboard extension')
        extension = plistlib.loads(archive.read(extensions[0]))
        if extension['CFBundleIdentifier'] != 'com.tuigo.keyboard' or extension['CFBundleShortVersionString'] != version:
            raise ValueError('Keyboard extension version or bundle ID mismatch')
        if extension['NSExtension']['NSExtensionPointIdentifier'] != 'com.apple.keyboard-service':
            raise ValueError('Embedded extension is not a keyboard')
        if base + 'Assets.car' not in names:
            raise ValueError('Missing compiled app logo/icon assets')
        print(f'PASS IPA {path.name}: version {version}, app + keyboard extension + assets')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('command', choices=['metadata', 'apk', 'ipa', 'check-ipa'])
    parser.add_argument('--tag')
    parser.add_argument('--source', type=Path)
    parser.add_argument('--output', type=Path, default=ROOT)
    args = parser.parse_args()
    values = metadata(args.tag)
    if args.command == 'metadata':
        print(json.dumps(values, ensure_ascii=False))
        if os.getenv('GITHUB_OUTPUT'):
            with open(os.environ['GITHUB_OUTPUT'], 'a', encoding='utf-8') as output:
                for key, value in values.items():
                    output.write(f'{key}={value}\n')
        return
    if args.source is None:
        parser.error('--source is required for packaging/inspection')
    if args.command == 'check-ipa':
        check_ipa(args.source, values['version'])
        return
    args.output.mkdir(parents=True, exist_ok=True)
    if args.command == 'apk':
        if not args.source.is_file() or args.source.suffix != '.apk':
            raise ValueError('Expected an explicit built APK path')
        manifest = args.source.parent / 'output-metadata.json'
        info = json.loads(manifest.read_text())
        if info['applicationId'] != 'com.tuigo.tiengviet' or len(info['elements']) != 1:
            raise ValueError('Unexpected APK output')
        element = info['elements'][0]
        if element['versionName'] != values['version'] or element['outputFile'] != args.source.name:
            raise ValueError('APK metadata does not match version/source')
        target = args.output / values['apk_name']
        shutil.copy2(args.source, target)
        print(f'PASS APK {target.name}: version {values["version"]}')
    else:
        apps = list(args.source.glob('*.app'))
        if len(apps) != 1:
            raise ValueError('Expected exactly one built .app')
        target = args.output / values['ipa_name']
        # zip preserves Unicode app names and includes app/embedded-extension files.
        with zipfile.ZipFile(target, 'w', compression=zipfile.ZIP_DEFLATED) as archive:
            for file in sorted(apps[0].rglob('*')):
                if file.is_file():
                    archive.write(file, 'Payload/' + file.relative_to(args.source).as_posix())
        check_ipa(target, values['version'])


if __name__ == '__main__':
    main()
