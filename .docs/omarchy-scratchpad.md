# Recommended Omarchy scratchpad setup

This setup starts Omaclip with Hyprland, waits until the selected network
interface is usable, and moves the Omaclip window to the `scratchpad` special
workspace without switching to it. After an entry is copied from Omaclip, the
copy hook closes the scratchpad and pastes the entry into the previously focused
application.

## Requirements

Install Omaclip and ensure `wtype`, `jq`, and `ip` are available. Omarchy
provides the Hyprland Lua configuration and `hyprctl` used by this setup.

## Find the network interface

Use the interface carrying the default route:

```bash
ip -o route get 1.1.1.1 | awk '{for (i = 1; i <= NF; i++) if ($i == "dev") {print $(i + 1); exit}}'
```

This only asks the kernel which route it would use; it does not contact
`1.1.1.1`. The output may be similar to `enp7s0` for Ethernet or `wlan0` for
Wi-Fi. Use that value for `mdns_interface` below. If clipboard peers are on a
network other than the default route, choose the interface attached to that
network instead.

## Add the scratchpad launcher

Create `~/.config/hypr/openandmovetoscratchpad.sh`:

```bash
#!/usr/bin/env bash

INTERFACE=""
if [ "${1:-}" = "--wait-for-interface" ]; then
  if [ "$#" -lt 3 ]; then
    echo "Error: --wait-for-interface requires an interface and a command" >&2
    exit 1
  fi

  INTERFACE="$2"
  shift 2
fi

if [ "$#" -eq 0 ]; then
  echo "Error: no command specified" >&2
  echo "Usage: $0 [--wait-for-interface <interface>] <command> [args...]" >&2
  exit 1
fi

while [ -n "$INTERFACE" ]; do
  if [ "$(cat "/sys/class/net/$INTERFACE/operstate" 2>/dev/null)" = "up" ] \
    && ip -o address show dev "$INTERFACE" scope global -tentative 2>/dev/null | grep -q . \
    && ip -o route show default dev "$INTERFACE" 2>/dev/null | grep -q '^default '; then
    break
  fi
  sleep 1
done

"$@" >/dev/null 2>&1 &
PID=$!

TIMEOUT=15
ELAPSED=0
while ! hyprctl clients -j | jq -e ".[] | select(.pid == $PID)" >/dev/null 2>&1; do
  sleep 0.5
  ELAPSED=$((ELAPSED + 1))
  if [ "$ELAPSED" -ge $((TIMEOUT * 2)) ]; then
    echo "Error: window for PID $PID never appeared" >&2
    exit 1
  fi
done

hyprctl dispatch "hl.dsp.window.move({ workspace = \"special:scratchpad\", follow = false, window = \"pid:$PID\" })"
```

Make it executable:

```bash
chmod +x ~/.config/hypr/openandmovetoscratchpad.sh
```

The launcher waits for the configured interface to be operational, have a
non-tentative global address, and carry the default route. It then starts
Omaclip, finds the window belonging to that exact process, and moves it to the
scratchpad without changing the active workspace.

## Start Omaclip from Hyprland

Add this to `~/.config/hypr/autostart.lua`, replacing `enp7s0` with the interface
found above:

```lua
local scratchpad_script = (os.getenv("HOME") or "") .. "/.config/hypr/openandmovetoscratchpad.sh"
local mdns_interface = "enp7s0"
local copy_hook = [[hyprctl dispatch "hl.dsp.workspace.toggle_special(\"scratchpad\")" && sleep 0.2 && wtype -M ctrl -M shift -P v -p v -m shift -m ctrl]]

local omaclip_command = table.concat({
  "omaclip",
  "--peers-mdns-interface=" .. o.shell_quote(mdns_interface),
  "--copy-hook=" .. o.shell_quote(copy_hook),
  "--clipboard-max-history=100",
}, " ")

o.exec_on_start(
  o.shell_quote(scratchpad_script)
    .. " --wait-for-interface " .. o.shell_quote(mdns_interface)
    .. " " .. omaclip_command
)
```

`--clipboard-max-history=100` raises the local in-memory history limit from 50
to 100 entries. The option is named `--clipboard-max-history`; there is no
`--max-clipboard-entries` option.

Reload and validate the Hyprland configuration:

```bash
hyprctl reload
hyprctl configerrors
```

After the next login, Omaclip starts when the network interface becomes usable
and its window is moved to `special:scratchpad`. Use your normal scratchpad
binding to show it. Selecting an entry runs the copy hook: the scratchpad closes,
focus returns to the previous application, and `wtype` sends `Ctrl+Shift+V`.

The `sleep 0.2` delay allows the previous window to regain focus before `wtype`
sends the paste shortcut. Applications that use `Ctrl+V` instead can replace
the `wtype` arguments accordingly.
