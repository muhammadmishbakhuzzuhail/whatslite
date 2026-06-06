package storage

// scheduled.go — pesan terjadwal + pengingat pesan (client-side, lean).
// Dieksekusi oleh ticker app saat waktu tiba (hanya selagi app hidup; catch-up
// saat app dibuka lagi).

import "context"

// Scheduled = satu pesan terjadwal.
type Scheduled struct {
	ID       string `json:"id"`
	ChatJID  string `json:"chatJid"`
	ChatName string `json:"chatName"`
	Text     string `json:"text"`
	SendAt   int64  `json:"sendAt"`
	Created  int64  `json:"created"`
}

func (s *Store) AddScheduled(ctx context.Context, m Scheduled) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO scheduled (id, chat_jid, chat_name, text, send_at, created) VALUES (?,?,?,?,?,?)`,
		m.ID, m.ChatJID, m.ChatName, m.Text, m.SendAt, m.Created)
	return err
}

func (s *Store) DeleteScheduled(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM scheduled WHERE id=?`, id)
	return err
}

// ListScheduled mengembalikan semua terjadwal (urut waktu kirim).
func (s *Store) ListScheduled(ctx context.Context) ([]Scheduled, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, chat_jid, chat_name, text, send_at, created FROM scheduled ORDER BY send_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Scheduled{}
	for rows.Next() {
		var m Scheduled
		if rows.Scan(&m.ID, &m.ChatJID, &m.ChatName, &m.Text, &m.SendAt, &m.Created) == nil {
			out = append(out, m)
		}
	}
	return out, rows.Err()
}

// DueScheduled mengembalikan yang send_at <= now (siap dikirim).
func (s *Store) DueScheduled(ctx context.Context, now int64) ([]Scheduled, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, chat_jid, chat_name, text, send_at, created FROM scheduled WHERE send_at <= ? ORDER BY send_at ASC`, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Scheduled{}
	for rows.Next() {
		var m Scheduled
		if rows.Scan(&m.ID, &m.ChatJID, &m.ChatName, &m.Text, &m.SendAt, &m.Created) == nil {
			out = append(out, m)
		}
	}
	return out, rows.Err()
}

// Reminder = pengingat pada sebuah pesan/chat.
type Reminder struct {
	ID       string `json:"id"`
	ChatJID  string `json:"chatJid"`
	ChatName string `json:"chatName"`
	MsgID    string `json:"msgId"`
	Note     string `json:"note"`
	RemindAt int64  `json:"remindAt"`
}

func (s *Store) AddReminder(ctx context.Context, r Reminder) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO reminders (id, chat_jid, chat_name, msg_id, note, remind_at) VALUES (?,?,?,?,?,?)`,
		r.ID, r.ChatJID, r.ChatName, r.MsgID, r.Note, r.RemindAt)
	return err
}

func (s *Store) DeleteReminder(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM reminders WHERE id=?`, id)
	return err
}

func (s *Store) ListReminders(ctx context.Context) ([]Reminder, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, chat_jid, chat_name, msg_id, note, remind_at FROM reminders ORDER BY remind_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Reminder{}
	for rows.Next() {
		var r Reminder
		if rows.Scan(&r.ID, &r.ChatJID, &r.ChatName, &r.MsgID, &r.Note, &r.RemindAt) == nil {
			out = append(out, r)
		}
	}
	return out, rows.Err()
}

func (s *Store) DueReminders(ctx context.Context, now int64) ([]Reminder, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, chat_jid, chat_name, msg_id, note, remind_at FROM reminders WHERE remind_at <= ? ORDER BY remind_at ASC`, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Reminder{}
	for rows.Next() {
		var r Reminder
		if rows.Scan(&r.ID, &r.ChatJID, &r.ChatName, &r.MsgID, &r.Note, &r.RemindAt) == nil {
			out = append(out, r)
		}
	}
	return out, rows.Err()
}
