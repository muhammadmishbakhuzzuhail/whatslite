// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail

package storage

// calls.go — log panggilan masuk. whatsmeow tak punya media call (WebRTC), jadi
// fitur "telepon" hanya: catat panggilan masuk + tolak. status: missed|rejected.

import (
	"context"
	"time"
)

// Call = satu baris log panggilan.
type Call struct {
	ID     string    `json:"id"`
	JID    string    `json:"jid"`
	Name   string    `json:"name"`
	Video  bool      `json:"video"`
	Group  bool      `json:"group"`
	Status string    `json:"status"` // missed | rejected
	Time   time.Time `json:"-"`
	TS     int64     `json:"ts"`
}

// SaveCall menyimpan/menimpa entri log panggilan.
func (s *Store) SaveCall(ctx context.Context, c Call) error {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO calls (id, jid, name, video, grp, status, ts)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET status=excluded.status, name=excluded.name`,
		c.ID, c.JID, c.Name, b2i(c.Video), b2i(c.Group), c.Status, c.Time.Unix())
	return err
}

// SetCallStatus memperbarui status panggilan (mis. → rejected setelah ditolak).
func (s *Store) SetCallStatus(ctx context.Context, id, status string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE calls SET status=? WHERE id=?`, status, id)
	return err
}

// ListCalls mengembalikan log panggilan terbaru (maks 200).
func (s *Store) ListCalls(ctx context.Context) ([]Call, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, jid, name, video, grp, status, ts FROM calls ORDER BY ts DESC LIMIT 200`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Call{}
	for rows.Next() {
		var c Call
		var video, grp int
		var ts int64
		if err := rows.Scan(&c.ID, &c.JID, &c.Name, &video, &grp, &c.Status, &ts); err != nil {
			return nil, err
		}
		c.Video, c.Group, c.TS = video == 1, grp == 1, ts
		out = append(out, c)
	}
	return out, rows.Err()
}
