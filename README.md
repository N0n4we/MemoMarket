# MemoMarket

Memo Manager & Market for [MemoChat](https://github.com/user/MemoChat). Create, manage, and share rule packs for MemoChat's memory system.

## Features

- **Rule Pack Manager** — Create, edit, delete, and organize rule packs with system prompts and memo rules
- **AI-Powered Generation** — Describe what you need and let AI generate a complete rule pack
- **MemoChat Compatible** — Import rules from MemoChat, export packs in MemoChat-compatible format
- **Search & Filter** — Search packs by name, description, author, or tags; filter by tag
- **Install Tracking** — Mark packs as installed to track which ones you're using
- **Import/Export** — Full MemoMarket pack format and MemoChat rules format supported
- **Settings Panel** — Configure OpenAI-compatible API endpoint for AI generation

## Tech Stack

- [Tauri v2](https://v2.tauri.app/) (Rust)
- [Vue 3](https://vuejs.org/) + TypeScript
- [Vite](https://vite.dev/)

## Development

```bash
# Install dependencies
pnpm install

# Start dev server (NixOS)
nix-shell --run "pnpm tauri dev"

# Build
pnpm tauri build
```

## Rule Pack Format

A rule pack contains:

| Field | Description |
|-------|-------------|
| Name | Pack name |
| Description | What the pack does |
| Author | Creator name |
| Version | Semantic version |
| System Prompt | System prompt for the AI |
| Rules | Array of {title, updateRule} pairs |
| Tags | Categorization tags |

## License

MIT
