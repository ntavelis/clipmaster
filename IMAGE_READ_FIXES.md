# Image Read Fixes — Session Notes

## Problem

On macOS, copying large images (e.g. full-screen screenshots) silently fails — nothing appears in clipboard history.

## Root Causes Identified

### 1. Silent error swallowing in Darwin backend

`darwin_clipboard.go` `GetImage` was swallowing errors from `readClipboardAs`. If PNGf read timed out, it fell through to JPEG (also failed), then returned `nil, nil` — no error at all. The monitor saw no error and no data, so it silently ignored the image.

**Fixed:** `GetImage` now returns errors from `readClipboardAs` instead of swallowing them. Consistent with how Wayland and xclip backends already behave. xsel is N/A (no image support).

### 2. No error logging in monitor

`readClipboard` in `monitor.go` never logged `imgErr`. Even if the backend returned an error, it was invisible.

**Fixed:** Added `m.log.Error("clipboard image read failed", "error", imgErr)` when `imgErr != nil`.

### 3. Shared 2-second timeout for text + image reads

`readClipboard` creates a single `context.WithTimeout(parent, cmdTimeout)` (2s) that covers both `GetText` and `GetImage` sequentially. If text takes 0.5s, image only has 1.5s left. For large screenshots going through osascript + temp file, that's not enough.

**Not yet implemented.** Plan: use separate timeouts derived from `parent`:

```go
textCtx, textCancel := context.WithTimeout(parent, cmdTimeout)       // 2s
defer textCancel()
text, err := m.reader.GetText(textCtx)

imgCtx, imgCancel := context.WithTimeout(parent, imgCmdTimeout)      // 10s
defer imgCancel()
imgData, imgErr := m.reader.GetImage(imgCtx)
```

Each child context is independent — cancelling `textCancel` does not affect `imgCtx`. Only cancelling `parent` stops both.

A 10s image timeout does not bypass size limits. A 15 MB image that takes 5s to read will still be rejected by the `maxPngImageMB`/`maxNonPngImageMB` size check.

### 4. Hash mismatch when preferring JPEG in Darwin GetImage

If `GetImage` is changed to prefer JPEG over PNG, `CopyItem` causes a re-add loop:

1. `CopyItem` decodes base64 image data (already normalized to PNG at ingestion)
2. Converts via `toPNG`, writes PNG to clipboard
3. Sets `lastSeenHash = sha256Hex(pngBytes)`
4. Next poll: `GetImage` returns JPEG bytes, hash differs from `lastSeenHash`
5. Image gets re-added as a duplicate

Hashing before `toPNG` doesn't help either — the entry's `ImageData` is already PNG (normalized at ingestion), so decoded bytes are PNG, not the original JPEG that `GetImage` will return.

**Not yet implemented.** Plan: add an `ImageHash` field to `ClipboardEntry` that stores the hash of the raw bytes from `GetImage` at ingestion time. Then `CopyItem` sets `lastSeenHash = entry.ImageHash` instead of hashing the converted output.
