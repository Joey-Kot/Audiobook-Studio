# Running Audiobook-Studio

Audiobook-Studio is a Wails desktop app. It uses the operating system WebView runtime for the GUI.

## Windows

Windows builds require Microsoft Edge WebView2 Runtime.

Windows 11 usually includes it. Some Windows 10, Server, LTSC, or stripped-down systems may not.

Run `Audiobook-Studio.exe` directly. The Windows executable is built with Wails' WebView2 `download` strategy, so it should prompt the user to download WebView2 if the runtime is missing.

You can also install it manually from:

https://developer.microsoft.com/microsoft-edge/webview2/

## Linux

Linux builds require GTK 3 and WebKitGTK 4.0 runtime libraries. Many desktop distributions already include GTK, but WebKitGTK may need to be installed.

Use `check-linux-runtime.sh` from the release package to check the current machine.

Common packages:

- Ubuntu/Debian: `sudo apt-get install libgtk-3-0 libwebkit2gtk-4.0-37`
- Fedora: `sudo dnf install gtk3 webkit2gtk4.0`
- Arch: `sudo pacman -S gtk3 webkit2gtk-4.1`

Package names can vary by distribution release.

## macOS

macOS builds use the system WebKit framework. No separate WebView runtime is normally required.
