# Clipmaster

A lightweight desktop clipboard manager built with Wails and Vue 3. It tracks your clipboard history, lets you browse and re-copy past items, and is designed for keyboard-first workflows.

- In-memory clipboard history (text only, up to 50 items)
- Keyboard navigation with shortcuts for quick copying (Ctrl+1..9)
- Expandable entries for viewing long text
- Live Omarchy theme support — colors update automatically when you switch themes
- Multi-machine sync via WebRTC (planned)

## Live Development

To run in live development mode, run `wails dev` in the project directory. This will run a Vite development
server that will provide very fast hot reload of your frontend changes. If you want to develop in a browser
and have access to your Go methods, there is also a dev server that runs on http://localhost:34115. Connect
to this in your browser, and you can call your Go code from devtools.

## Building

To build a redistributable, production mode package, use `wails build`.
