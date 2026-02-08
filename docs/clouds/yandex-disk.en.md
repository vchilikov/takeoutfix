# Yandex Disk Upload Guide (English)

Back to cloud index: [docs/clouds/README.md](README.md)

Русская версия: [yandex-disk.ru.md](yandex-disk.ru.md)

## Scope

Use Yandex Disk if you want a familiar web interface, shared folders, and simple desktop sync on macOS/Windows.

## Web Upload (macOS, Windows, Linux)

1. Open [Yandex Disk web](https://disk.yandex.com) and sign in.
2. Create a destination folder, for example `GooglePhotos-Migration`.
3. Open `takeoutfix-extracted/Takeout` on your computer.
4. Drag the full content of `Takeout` into the destination folder in your browser.
5. Wait until all uploads finish, then refresh and confirm file counts.

Suggested screenshot: `docs/images/10_yandex_web_upload.png`.

## Desktop Sync on macOS

1. Install Yandex Disk desktop app from the official Yandex website.
2. Sign in with your Yandex account.
3. Open your local Yandex Disk sync folder in Finder.
4. Copy the content of `takeoutfix-extracted/Takeout` into that sync folder.
5. Wait until sync status shows that all files are uploaded.

## Desktop Sync on Windows

1. Install Yandex Disk desktop app from the official Yandex website.
2. Sign in and complete first-time setup.
3. Open the local Yandex Disk folder in File Explorer.
4. Copy the content of `takeoutfix-extracted/Takeout` into the sync folder.
5. Wait until the app reports sync is complete.

Suggested screenshot: `docs/images/11_yandex_desktop_sync_windows.png`.

## Linux Fallback (Web-Only)

For non-technical migration, use the web upload flow on Linux. This is the safest and most predictable path across distributions.

## Post-Upload Verification Checklist

1. Open at least 20 random photos and 5 random videos.
2. Check that capture dates are correct in timeline view.
3. Check that location is present on files that originally had geolocation.
4. Confirm that albums/folders are still structured as expected.
5. Confirm upload completion in Yandex Disk activity/history.

## Typical Mistakes and Quick Fixes

- Mistake: Uploading ZIP archives instead of processed files.
  - Fix: Upload `takeoutfix-extracted/Takeout` only.
- Mistake: Flattening folders during drag-and-drop.
  - Fix: Recreate a root folder and upload while preserving hierarchy.
- Mistake: Deleting local files too early.
  - Fix: Keep local processed copy until verification is complete.
- Mistake: Browser upload interrupted.
  - Fix: Re-run upload for missing folders and verify file counts.
