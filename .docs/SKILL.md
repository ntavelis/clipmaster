---
name: omaclip
description: Use Omaclip from an agent: add text or images to clipboard history and configure post-copy hooks.
---

# Omaclip

Use this skill when asked to place content in Omaclip history or configure `--copy-hook`.

## Add content to history

Omaclip must be running. On Wayland, copy supported content to the system clipboard and let Omaclip observe it.

Text:

```bash
printf '%s' "$CONTENT" | wl-copy --type text/plain
```

Text-file contents:

```bash
wl-copy --type text/plain < /path/to/file
```

PNG image:

```bash
wl-copy --type image/png < /path/to/image.png
```

For several history entries, use one `wl-copy` per entry and wait between writes:

```bash
wl-copy --type text/plain < first.txt
sleep 3
wl-copy --type text/plain < second.txt
sleep 3
```

The wait safely covers the default two-second polling fallback. The last item remains on the system clipboard. Omaclip may deduplicate identical content.

Do not use `text/uri-list` for text or source files. It copies file references, not contents. Omaclip ignores non-image file references because history supports text and images, not arbitrary files.

Inspect the current clipboard offer with:

```bash
wl-paste --list-types
wl-paste --no-newline
```

This verifies the system clipboard, not the complete Omaclip history.

## Copy hooks

`--copy-hook` runs after Omaclip successfully copies a selected local or remote item back to the system clipboard. Adding content with `wl-copy` does not trigger it.

The hook:

- runs asynchronously through `sh -c`;
- does not block copying;
- logs failures;
- receives no item arguments or dedicated environment variables;
- may read the clipboard, but that can race with later clipboard changes.

Configure it with `--copy-hook='<command>'` or `OMACLIP_COPY_HOOK`.

Notification:

```bash
omaclip --copy-hook='notify-send "Omaclip" "Item copied"'
```

Hyprland scratchpad: close it, wait for focus to return, then paste with `Ctrl+Shift+V`:

```bash
omaclip --copy-hook='hyprctl dispatch "hl.dsp.workspace.toggle_special(\"scratchpad\")" && sleep 0.2 && wtype -M ctrl -M shift -P v -p v -m shift -m ctrl'
```

Use `Ctrl+V` instead when the target application expects ordinary paste. Avoid hooks that upload or log clipboard contents, recursively rewrite the clipboard, or perform destructive actions.
