# OneDrive Upload Guide

Back to cloud index: [docs/clouds/README.md](README.md)

## Scope

Use OneDrive if you want tight Microsoft 365 integration and native Windows synchronization.

## Web Upload (macOS, Windows, Linux)

1. Open [OneDrive Web](https://onedrive.live.com) and sign in.
2. Create a destination folder, for example `GooglePhotos-Migration`.
3. Open `takeoutfix-extracted/Takeout` locally.
4. Upload the full folder content through browser upload or drag-and-drop.
5. Wait for completion and verify uploaded file count.

Suggested screenshot: `docs/images/14_onedrive_web_upload.png`.

## Desktop Sync on macOS

1. Install the OneDrive app for macOS and sign in.
2. Open the local OneDrive folder in Finder.
3. Copy the content of `takeoutfix-extracted/Takeout` into OneDrive.
4. Wait for sync completion in the OneDrive menu status.
5. Confirm files in OneDrive web.

## Desktop Sync on Windows

1. Open OneDrive and sign in with your Microsoft account.
2. Open your local OneDrive folder in File Explorer.
3. Copy the content of `takeoutfix-extracted/Takeout` into OneDrive.
4. Wait until OneDrive reports everything is up to date.
5. Confirm files in OneDrive web.

Suggested screenshot: `docs/images/15_onedrive_desktop_sync_windows.png`.

## Linux Fallback (Web-Only)

For non-technical users, use OneDrive web upload on Linux because there is no mainstream official desktop sync client.

## Post-Upload Verification Checklist

1. Open at least 20 random photos and 5 random videos.
2. Check date ordering in OneDrive/Photos views.
3. Check that folders are preserved as uploaded.
4. Confirm upload completion in recent activity.
5. Keep local processed copy until verification is done.

## Typical Mistakes and Quick Fixes

- Mistake: Uploading to the wrong Microsoft account.
  - Fix: Verify account identity before starting upload.
- Mistake: Partial upload due to unstable connection.
  - Fix: Re-upload missing folders and compare counts.
- Mistake: Renaming many folders mid-upload.
  - Fix: Let upload complete first, then rename if needed.
- Mistake: Deleting local copy immediately.
  - Fix: Keep local backup until a full quality check is done.
