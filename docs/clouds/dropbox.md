# Dropbox Upload Guide

Back to cloud index: [docs/clouds/README.md](README.md)

## Scope

Use Dropbox if you need broad cross-platform support, easy sharing, and stable desktop sync.

## Web Upload (macOS, Windows, Linux)

1. Open [Dropbox Web](https://www.dropbox.com/home) and sign in.
2. Create a destination folder, for example `GooglePhotos-Migration`.
3. Open `takeoutfix-extracted/Takeout` on your computer.
4. Use Upload or drag-and-drop to send all files and folders.
5. Wait for completion and verify the total number of uploaded items.

Suggested screenshot: `docs/images/12_dropbox_web_upload.png`.

## Desktop Sync on macOS

1. Install Dropbox desktop app and sign in.
2. Open your local Dropbox folder in Finder.
3. Copy the content of `takeoutfix-extracted/Takeout` into Dropbox.
4. Keep Dropbox running until sync status is complete.
5. Confirm files appear in Dropbox web.

Suggested screenshot: `docs/images/13_dropbox_desktop_sync_mac.png`.

## Desktop Sync on Windows

1. Install Dropbox desktop app and sign in.
2. Open your local Dropbox folder in File Explorer.
3. Copy the content of `takeoutfix-extracted/Takeout` into Dropbox.
4. Wait until sync icons show completion.
5. Confirm files appear in Dropbox web.

## Linux Fallback (Web-Only)

If desktop sync is unavailable or unstable on your Linux setup, use the web upload flow as the reliable fallback.

## Post-Upload Verification Checklist

1. Open at least 20 random photos and 5 random videos.
2. Confirm timeline ordering by capture date.
3. Verify folder hierarchy matches your local processed copy.
4. Check that uploads are fully complete in Dropbox activity.
5. Keep local copy until you finish verification.

## Typical Mistakes and Quick Fixes

- Mistake: Uploading original Takeout ZIP files.
  - Fix: Upload content from `takeoutfix-extracted/Takeout` only.
- Mistake: Moving files before Dropbox finishes sync.
  - Fix: Wait for full sync, then reorganize only if needed.
- Mistake: Browser tab closed during upload.
  - Fix: Reopen Dropbox web and re-upload missing folders.
- Mistake: Not enough Dropbox quota.
  - Fix: Check available storage before starting large uploads.
