# TakeoutFix

TakeoutFix helps you restore Google Photos Takeout metadata and write it back into your photos and videos.

It handles common Takeout export issues:

- reads date, GPS, and description from metadata `.json` sidecars;
- supports `.supplemental-metadata.json` sidecars (including truncated variants);
- fixes incorrect media file extensions;
- adds `OffsetTime*` (timezone offset) when it can be computed from GPS;
- supports repeated runs (`fix` is idempotent);
- removes only matched `.json` files in `clean-json` mode.

## Requirements

- [Go](https://go.dev/) 1.25+
- [ExifTool](https://exiftool.org) available in `PATH`

Quick check:

```bash
exiftool -ver
go version
```

## Quick Start

1. Build the binary:

```bash
go build -o takeoutfix
```

2. Run metadata fix:

```bash
takeoutfix fix ~/Downloads/Takeout/Google\ Photos
```

3. Optionally remove matched JSON files:

```bash
takeoutfix clean-json ~/Downloads/Takeout/Google\ Photos
```

If your path contains spaces, use `\` escaping or quotes.

## Expected Input Structure

TakeoutFix expects a standard Google Takeout layout.  
Matching runs recursively inside the provided root path, including cases where media and sidecars are in different subfolders.

```text
Takeout/
  Google Photos/
    Album 1/
      IMG_0001.jpg
      IMG_0001.jpg.json
      IMG_0002.mp4
      IMG_0002.mp4.supplemental-metadata.json
```

## Commands

### `fix`

Matches media files with JSON sidecars, fixes file extensions, and writes metadata with ExifTool.

```bash
takeoutfix fix <path-to-google-photos-folder>
```

Example with logs:

```bash
takeoutfix fix <path> | tee takeoutfix.log
```

### `clean-json`

Removes only `.json` files that were successfully matched to media files.  
Unused/orphaned and ambiguous `.json` files are kept and logged.

```bash
takeoutfix clean-json <path-to-google-photos-folder>
```

## Behavior and Limitations

- If timezone offset cannot be computed (missing GPS or failed lookup), processing continues without `OffsetTime*`.
- For formats not directly writable by ExifTool, metadata is written into an `.xmp` sidecar.
- The project is designed for Google Photos Takeout structure and naming patterns.

## Local Verification

```bash
go test ./...
go vet ./...
go build ./...
```

## Releases

Releases are automated with GitHub Actions + GoReleaser.

To publish a new version:

```bash
git tag vX.Y.Z
git push --tags
```

After pushing a tag, GitHub creates a release with binaries for Linux/macOS/Windows (`amd64` + `arm64`) and a `checksums.txt` file.

Download releases: [https://github.com/vchilikov/takeout-fix/releases](https://github.com/vchilikov/takeout-fix/releases)
