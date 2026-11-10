-- Per-outbound-message `MessageContextInfo.messageSecret` keyed by the message
-- our stanza went out under. Required to decrypt addons that reference it
-- (poll votes, reactions, msmsg bot replies, edits).
CREATE TABLE msg_secrets (
    chat TEXT NOT NULL,
    sender TEXT NOT NULL,
    msg_id TEXT NOT NULL,
    device_id INTEGER NOT NULL DEFAULT 1,
    secret BLOB NOT NULL,
    -- Absolute unix-seconds retention deadline; 0 means never expire.
    expires_at BIGINT NOT NULL DEFAULT 0,
    -- Parent message event time (unix seconds); 0 means unknown.
    message_ts BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (chat, sender, msg_id, device_id)
);

-- The prune scans `device_id = ? AND expires_at <> 0 AND expires_at <= ?`.
-- Rows with expires_at = 0 never expire, so the index is partial over the
-- expirable set; device_id leads to keep the scan scoped to one account.
CREATE INDEX idx_msg_secrets_expires
    ON msg_secrets (device_id, expires_at)
    WHERE expires_at <> 0;
