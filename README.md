# TakeoutFix

TakeoutFix restores Google Photos Takeout metadata and writes it back into photo/video files.

## What it does

- Matches media files with Google Takeout JSON sidecars recursively under a root folder.
- Supports classic `*.json` and supplemental `*.supplemental-metadata.json` sidecars, including truncated variants.
- Handles common naming variants (`-edited`, `(1)`, random suffixes like `-abc12`, long-name truncation).
- Supports cross-folder matching when media and JSON live in different subfolders.
- Fixes media file extension using `exiftool -p '.$FileTypeExtension'`.
- Applies metadata via ExifTool (`Title`, `Description`, `AllDates`, GPS tags).
- Writes to `<media>.xmp` sidecar when the media extension is not directly writable.
- Keeps repeated runs safe (`fix` can be run multiple times).
- Removes only successfully matched JSON files in `clean-json`.

## Requirements

- [Go](https://go.dev/) `1.25+`
- [ExifTool](https://exiftool.org) available in `PATH`

Check:

```bash
go version
exiftool -ver
```

## Build

```bash
go build -o takeoutfix .
```

## Usage

```bash
takeoutfix <operation> <path>
```

Supported operations:

```bash
takeoutfix fix /path/to/Takeout/Google\ Photos
takeoutfix clean-json /path/to/Takeout/Google\ Photos
```

If the path contains spaces, use quotes or escaping.

## Command behavior

### `fix`

1. Scans all nested folders under the given root.
2. Matches each media file to the best JSON candidate.
3. Prints warnings for missing/unused/ambiguous matches.
4. Fixes extension if needed.
5. Applies metadata with ExifTool.

Example:

```bash
takeoutfix fix "/Users/me/Downloads/Takeout/Google Photos" | tee takeoutfix.log
```

### `clean-json`

- Runs the same scan/matching logic as `fix`.
- Removes only JSON files that were uniquely matched to media.
- Keeps orphaned and ambiguous JSON files untouched.

Example:

```bash
takeoutfix clean-json "/Users/me/Downloads/Takeout/Google Photos"
```

## Matching notes

- Matching is case-insensitive.
- `.xmp` files are ignored as input media.
- If a media file has multiple JSON candidates, it is marked ambiguous and skipped.
- If one global JSON candidate could match multiple media files, it is also treated as ambiguous.
- For `.mp4`, matcher also tries related stems with `.jpg`, `.jpeg`, `.heic`.

## Metadata notes and limitations

- `OffsetTime*` tags are intentionally not written.
- Unsupported writable formats are handled through `.xmp` sidecars.
- This tool is tailored for Google Photos Takeout naming/layout patterns.

## Troubleshooting

- `exiftool: command not found`
  - Install ExifTool and ensure it is in `PATH`.
- Many `unused json kept` warnings
  - Usually means unmatched/orphan sidecars or ambiguous matches in your export.
- Unexpected rename
  - TakeoutFix trusts ExifTool-detected type and may rename the extension when needed.

## Local verification

```bash
go test ./...
go vet ./...
go build ./...
```

## Releases

Releases are automated with GitHub Actions + GoReleaser.

After each merge into `main`, CI creates the next tag in `YYYY.MM.N` format and triggers release publishing automatically.

Manual fallback:

```bash
git tag 2026.02.1
git push origin 2026.02.1
```

Tag format must be `YYYY.MM.N` and point to current `main` HEAD.

Artifacts include Linux/macOS/Windows binaries (`amd64` + `arm64`) and `checksums.txt`.

Download releases: [https://github.com/vchilikov/takeout-fix/releases](https://github.com/vchilikov/takeout-fix/releases)
