# iCloud Photos Upload Guide

Back to cloud index: [docs/clouds/README.md](README.md)

## Scope

Use iCloud Photos if your main devices are Apple devices and you want photos synchronized into the Apple Photos ecosystem.

## Web Upload (macOS, Windows, Linux)

1. Open [iCloud Photos Web](https://www.icloud.com/photos) and sign in with Apple ID.
2. Create a destination flow in your library (for example, upload by year/month batches).
3. Open `takeoutfix-extracted/Takeout` on your computer.
4. Use the upload button and select files/folders in manageable batches.
5. Wait for each batch to complete before starting the next one.

Suggested screenshot: `docs/images/16_icloud_web_upload.png`.

## Desktop Sync on macOS

1. Open Photos on macOS and ensure iCloud Photos is enabled in settings.
2. Choose `File -> Import`.
3. Import files from `takeoutfix-extracted/Takeout`.
4. Keep Mac online and Photos open until iCloud sync completes.
5. Verify uploaded items on iCloud Photos web.

## Desktop Sync on Windows

1. Install iCloud for Windows from Microsoft Store.
2. Sign in with Apple ID and enable Photos sync.
3. Open the iCloud Photos location in File Explorer.
4. Copy media from `takeoutfix-extracted/Takeout` into the upload area.
5. Wait for iCloud upload to complete and verify on iCloud web.

Suggested screenshot: `docs/images/17_icloud_sync_windows_or_macos.png`.

## Linux Fallback (Web-Only)

Use iCloud web upload on Linux. There is no official iCloud desktop sync app for Linux.

## Post-Upload Verification Checklist

1. Open at least 20 random photos and 5 random videos.
2. Verify capture dates in Apple Photos/iCloud timeline.
3. Verify location metadata on samples where location should exist.
4. Confirm media is visible on another Apple device.
5. Keep local processed copy until checks are complete.

## Typical Mistakes and Quick Fixes

- Mistake: Uploading too many files in one web batch.
  - Fix: Upload in smaller batches to reduce failed transfers.
- Mistake: iCloud Photos disabled on device.
  - Fix: Enable iCloud Photos before import/upload.
- Mistake: Upload interrupted when device sleeps.
  - Fix: Keep device awake and connected during upload.
- Mistake: Immediate cleanup after first sync.
  - Fix: Verify across devices before deleting local files.
