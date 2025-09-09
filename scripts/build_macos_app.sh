#!/usr/bin/env bash
set -euo pipefail
VERSION=${1:-dev}
APP=bar-extract
BIN=dist/$APP
if [[ ! -f "$BIN" ]]; then
  echo "Binary $BIN missing. Run 'make build' first." >&2
  exit 1
fi
BUNDLE=dist/$APP.app
CONTENTS=$BUNDLE/Contents
MACOS=$CONTENTS/MacOS
mkdir -p "$MACOS"
cat > "$CONTENTS/Info.plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleDevelopmentRegion</key><string>en</string>
  <key>CFBundleExecutable</key><string>$APP</string>
  <key>CFBundleIdentifier</key><string>com.example.$APP</string>
  <key>CFBundleInfoDictionaryVersion</key><string>6.0</string>
  <key>CFBundleName</key><string>$APP</string>
  <key>CFBundlePackageType</key><string>APPL</string>
  <key>CFBundleShortVersionString</key><string>${VERSION}</string>
  <key>CFBundleVersion</key><string>${VERSION}</string>
  <key>LSMinimumSystemVersion</key><string>11.0</string>
  <key>CFBundleDocumentTypes</key>
  <array>
    <dict>
      <key>CFBundleTypeName</key><string>BAR Archive</string>
      <key>CFBundleTypeExtensions</key><array><string>bar</string></array>
      <key>CFBundleTypeRole</key><string>Viewer</string>
      <key>LSHandlerRank</key><string>Default</string>
    </dict>
  </array>
</dict>
</plist>
EOF
cp "$BIN" "$MACOS/$APP"
chmod +x "$MACOS/$APP"
 echo "Created $BUNDLE"
