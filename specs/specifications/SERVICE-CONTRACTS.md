# Mock service contracts

Each method returns a `Result`-style response and supports deterministic scripted latency/error cases. Define Rust async or GPUI Task-compatible signatures after pinning GPUI runtime.

| Service | Operations | Fake scenarios |
|---|---|---|
| AuthService | start_qr_pairing, start_phone_linking, poll_link_status, logout | pending, expired, success, invalid code, offline |
| ChatService | list_chats, get_messages, send, edit, delete, forward, react, search | normal, empty, retryable failure, partial sync |
| ContactService | list_contacts, search_contacts, get_contact, block | found, empty, blocked |
| GroupService | create, update, members, permissions, invite | no members, permission denied |
| MediaService | choose_files, upload, download, media_index | progress, canceled, failed, oversized |
| CallService | history, place, accept, reject, hangup, set_mute, set_video, share | ringing, connected, rejected, device unavailable |
| SettingsStore | load, update, set_theme | light, dark, persistence error |
| NotificationService | show_in_app, dismiss | foreground, muted, focus mode |

Fixtures: contacts (`maya`, `budi`, `rizky`), groups (`product-team`, `family`), messages (`m001` ...), attachment (`brief.pdf`). Use deterministic timestamp ISO strings. Domain IDs must be stable across modules.

**Not real integration:** Linking, calls, sending, notifications and media are simulated and do not contact WhatsApp servers. Phone-number linking is companion-device linking, not SMS OTP login.
