# Security Policy

## Scope

This project stores your **WhatsApp session and end-to-end encryption keys** locally in
`~/.local/share/whatsapp-lite/` (SQLite). Vulnerabilities that leak or misuse this data are serious.

**In scope:**

- Leakage of local sessions/keys/messages (e.g. wrong file permissions, path traversal when saving media).
- Code execution from incoming message content (XSS in the WebView via messages/notifications/link previews).
- Leakage of secrets to logs or the network (unexpected telemetry).

**Out of scope:** the risk of Meta banning your account (that's a documented ToS consequence, not a bug),
and vulnerabilities in upstream whatsmeow/Wails (report those to the respective projects).

## How to report

**Do not open a public issue for security vulnerabilities.** Instead:

- Use **GitHub Security Advisories** (the *Security → Report a vulnerability* tab) on this repo, or
- Send a private email to the maintainer.

Include: reproduction steps, version/commit, and impact. Please allow a reasonable window for a fix
before publishing (coordinated disclosure).

## No warranty

See the [disclaimer in the README](./README.md#-disclaimer-read-first). The software is provided **without any warranty**.
Using an unofficial protocol carries a risk of being banned; use it at your own risk.
