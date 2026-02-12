# TakeoutFix

TakeoutFix is an interactive tool for regular users who want to move out of Google Photos without losing metadata.
It prepares Google Takeout archives and restores photo/video metadata such as capture date, location, and description.

## Quick Start (3 Steps)

1. Export your Google Photos archive from Google Takeout as ZIP files.
2. Put all `*.zip` files into one local folder.
3. Run TakeoutFix in that folder, then upload the processed output to your new storage.

## One-liner Installer + Runner (Recommended)

Run the command directly in the folder with your Takeout ZIP files.

macOS/Linux:

```bash
curl -fsSL https://github.com/vchilikov/takeout-fix/releases/latest/download/install.sh | sh
```

Windows (PowerShell):

```powershell
iwr -useb https://github.com/vchilikov/takeout-fix/releases/latest/download/install.ps1 | iex
```

The installer works in your current folder, detects OS/arch, downloads the latest release, verifies `checksums.txt`, ensures `exiftool` is available (best-effort auto-install), runs processing, and removes the downloaded runtime binary after completion.

## Run TakeoutFix Manually

### Option A: Download a ready binary (recommended)

Get the latest release for your OS from [GitHub Releases](https://github.com/vchilikov/takeout-fix/releases).

### Option B: Build from source

```bash
go build -o takeoutfix .
```

### Start processing

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

## Minimal Requirements

- Supported OS: macOS, Linux, Windows.
- `exiftool` must be available in `PATH`.
- If `exiftool` is missing, use the one-liner installer above or install it manually.
- You need enough free disk space for extraction.

## What You Will See in the Terminal

TakeoutFix runs in a guided sequence and prints clear stages:

- `TakeoutFix interactive mode`
- `Checking dependencies...`
- `Scanning ZIP archives in current folder...`
- `Validating ZIP integrity (all archives)...`
- `Checking available disk space...`
- `Extracting archives into: ...`
- `Applying metadata and cleaning matched JSON...`
- `Final summary`

If any archive is corrupt, processing stops before extraction.

Status in the final summary:

- `SUCCESS` - processing finished without hard file-level errors.
- `PARTIAL_SUCCESS` - some media failed metadata/extension processing. Re-run is recommended after fixing issues.

## Output: What to Upload

After a successful run (`Status: SUCCESS`):

- Processed files are in `./takeoutfix-extracted`.
- Upload `./takeoutfix-extracted/Takeout` to your new cloud storage.
- Original Takeout ZIP files are auto-removed according to the deletion mode below.

ZIP deletion mode:

- Normal mode (enough free disk): ZIPs are deleted only after successful metadata processing.
- Low-space mode (`required` does not fit, but `required with auto-delete` fits): ZIPs are deleted immediately after extraction.
- If final status is `PARTIAL_SUCCESS` in normal mode, ZIPs are kept for rerun.

Technical state is saved in `./.takeoutfix/state.json` so reruns can skip unchanged archives.

Exit codes:

- `0` - `SUCCESS`
- `2` - preflight failure
- `3` - runtime failure, including `PARTIAL_SUCCESS`

## Cloud Upload Guides

Use cloud-specific guides here: [docs/clouds/README.md](docs/clouds/README.md)

Yandex Disk Russian guide: [docs/clouds/yandex-disk.ru.md](docs/clouds/yandex-disk.ru.md)

## Troubleshooting

- **macOS: "takeoutfix" Not Opened / Apple could not verify**
  - This is normal for binaries downloaded from the internet. Remove the quarantine attribute and run again:
    ```bash
    xattr -d com.apple.quarantine ./takeoutfix
    ```
- `No ZIP archives found in current folder.`
  - Move all Takeout ZIPs to the top level of your working folder and run again.
- `Corrupt ZIP files found. Processing stopped.`
  - Re-download broken archive parts from Google Takeout, then rerun.
- `Missing dependencies: exiftool`
  - Run the one-liner installer command for your OS, or install `exiftool` manually and rerun.
- `Not enough disk space even with auto-delete enabled.`
  - Free up disk space and rerun.
- `Status: PARTIAL_SUCCESS`
  - Some files could not be processed. Fix the reported errors and rerun TakeoutFix.

## Developer Appendix (Optional)

```bash
make check
make check-all
```

This appendix is optional for end users and intended for contributors.
