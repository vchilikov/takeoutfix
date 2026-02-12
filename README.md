# TakeoutFix

Move your Google Photos library without losing important metadata.
TakeoutFix processes Google Takeout ZIP files and restores fields like capture date, location, and description.

[English](README.md) · [Русский](docs/README.ru.md) · [中文](docs/README.zh-CN.md) · [हिन्दी](docs/README.hi.md) · [Español](docs/README.es.md) · [Français](docs/README.fr.md) · [العربية](docs/README.ar.md) · [Deutsch](docs/README.de.md)

## Why TakeoutFix

- Keeps photo/video metadata that is often lost during migration.
- Works directly with standard Google Takeout ZIP exports.
- Guides you through a clear terminal flow from start to finish.
- Built for regular users: no scripting required.

## Quick Start (3 Steps)

1. Export your Google Photos archive from Google Takeout as ZIP files.
2. Put all `*.zip` files into one local folder.
3. Open a terminal in that folder and run the recommended command below.

## Run (Recommended)

Run directly in the folder with your Takeout ZIP files.

macOS/Linux:

```bash
curl -fsSL https://github.com/vchilikov/takeout-fix/releases/latest/download/install.sh | sh
```

Windows (PowerShell):

```powershell
iwr -useb https://github.com/vchilikov/takeout-fix/releases/latest/download/install.ps1 | iex
```

## Manual Run (Optional)

Use this only if you do not want the one-liner installer.

1. Download the latest binary for your OS from [GitHub Releases](https://github.com/vchilikov/takeout-fix/releases).
2. Run it in the folder that contains your Takeout ZIP files.

macOS/Linux:

```bash
./takeoutfix
./takeoutfix --workdir /path/to/folder
```

Windows (PowerShell):

```powershell
.\takeoutfix.exe
.\takeoutfix.exe --workdir C:\path\to\folder
```

## What You Get

After a successful run:

- Your processed media is ready in `./takeoutfix-extracted/Takeout`.
- Metadata is applied to supported photos and videos.
- JSON `Tags` are written to `Keywords` and `Subject`.
- If JSON capture timestamp is missing/invalid and filename starts with `YYYY-MM-DD HH.MM.SS`, date is restored from the filename.
- You can upload `./takeoutfix-extracted/Takeout` to your new storage.

## Common Issues

- `No ZIP archives found in current folder.`
  - TakeoutFix auto-detects already extracted Takeout content in the working folder.
  - If this message appears, either place Takeout ZIP parts in the folder root or run from a folder that contains extracted Takeout media.
- `Corrupt ZIP files found. Processing stopped.`
  - Re-download broken archive parts from Google Takeout, then rerun.
- `Missing dependencies: exiftool`
  - Use the recommended one-liner command above, or install `exiftool` manually and rerun.
- `Not enough disk space even with auto-delete enabled.`
  - Free up disk space and rerun.
- macOS says the app is not verified
  - Remove quarantine and run again:
    ```bash
    xattr -d com.apple.quarantine ./takeoutfix
    ```
