# Clipmaster — TODO

## 1. Cross-Platform Testing

- Test on other Linux distros beyond Arch
- Test on Darwin (macOS)
- Windows will not be supported

## 2. MCP Server for Agent Clipboard Access

- Create an MCP server that exposes clipboard entries (both images and text) to AI agents
- Should support returning local clipboard entries and remote peer clipboard entries
- **Open question:** Should we expose only the last clipboard item or the full history?
  - Exposing only the last item ensures the user deliberately copied something for the agent (no accidental data leakage)
  - Exposing more history could be useful but risks leaking sensitive copied content to the agent
  - Decision TBD
- **Image support is feasible:** MCP protocol supports `type: "image"` content blocks with base64-encoded data and MIME type. Text entries return as `type: "text"` blocks. Image clipboard support already exists in the app — just need to base64-encode the image bytes when serving through the MCP tool.

## 3. Persistent History with Encryption at Rest

- Optionally persist clipboard history to disk so it survives restarts
- Encrypt the stored history using the passphrase
- Should be opt-in so users who prefer ephemeral history can keep current behavior

## 4. Search and Filter

- Add a search/filter input to the history UI so users can quickly find past entries
- Filter should work across both local and remote clipboard histories
