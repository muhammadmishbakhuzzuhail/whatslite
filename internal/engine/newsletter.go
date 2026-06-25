// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail

package engine

// newsletter.go — WhatsApp Channels (newsletter). Daftar saluran yang diikuti,
// ambil pesannya (read-only), ikuti/berhenti via tautan undangan.

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

// ChannelInfo = metadata satu saluran (disederhanakan untuk frontend).
type ChannelInfo struct {
	JID         string
	Name        string
	Description string
	Subscribers int
	Picture     string // URL CDN langsung (atau "")
	Verified    bool
	Muted       bool
	Role        string // subscriber | admin | owner | guest
	Invite      string // kode undangan (utk tautan bagikan …/channel/<kode>)
}

// ChannelMsg = satu pesan saluran (read-only).
type ChannelMsg struct {
	ID        string
	ServerID  int64
	Text      string
	Kind      string
	Thumb     string
	Timestamp time.Time
	Views     int
}

// MarkChannelViewed menandai pesan saluran sudah dilihat (view-receipt).
func (e *Engine) MarkChannelViewed(ctx context.Context, jid string, serverIDs []int64) error {
	j, err := types.ParseJID(jid)
	if err != nil || len(serverIDs) == 0 {
		return err
	}
	ids := make([]types.MessageServerID, 0, len(serverIDs))
	for _, s := range serverIDs {
		ids = append(ids, types.MessageServerID(int(s)))
	}
	return e.Client.NewsletterMarkViewed(ctx, j, ids)
}

// CreateChannel membuat saluran (newsletter) baru; kembalikan JID.
func (e *Engine) CreateChannel(ctx context.Context, name, desc string, picture []byte) (string, error) {
	md, err := e.Client.CreateNewsletter(ctx, whatsmeow.CreateNewsletterParams{
		Name: name, Description: desc, Picture: picture,
	})
	if err != nil {
		return "", err
	}
	return md.ID.String(), nil
}

// ListChannels mengembalikan saluran yang sedang diikuti.
func (e *Engine) ListChannels(ctx context.Context) ([]ChannelInfo, error) {
	metas, err := e.Client.GetSubscribedNewsletters(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]ChannelInfo, 0, len(metas))
	for _, m := range metas {
		out = append(out, channelInfoFrom(m))
	}
	return out, nil
}

// RecommendedChannels mengambil saluran REKOMENDASI global (bukan hanya yang
// diikuti) lewat mex query "directory" WhatsApp via DangerousInternals — API ini
// tak punya wrapper publik di whatsmeow, jadi dipanggil mentah. Read-only.
// Format respons tak terdokumentasi → parsing defensif beberapa bentuk.
func (e *Engine) RecommendedChannels(ctx context.Context, query string) ([]ChannelInfo, error) {
	if e == nil || e.Client == nil {
		return nil, fmt.Errorf("not connected")
	}
	// queryRecommendedNewsletters = "7263823273662354" (output: xwa2_newsletters_recommended)
	input := map[string]any{"limit": 50, "country_codes": []string{}}
	if strings.TrimSpace(query) != "" {
		input["search_text"] = strings.TrimSpace(query) // best-effort filter
	}
	data, err := e.Client.DangerousInternals().SendMexIQ(ctx, "7263823273662354", map[string]any{"input": input})
	if err != nil {
		return nil, err
	}
	// Cari list NewsletterMetadata di dalam respons apa pun bentuknya.
	list := extractNewsletters(data)
	if len(list) == 0 {
		// surfacing utk debug: API ini undocumented; sertakan cuplikan mentah.
		snippet := string(data)
		if len(snippet) > 300 {
			snippet = snippet[:300]
		}
		return nil, fmt.Errorf("kosong/parse gagal; raw: %s", snippet)
	}
	out := make([]ChannelInfo, 0, len(list))
	for _, m := range list {
		if m != nil {
			out = append(out, channelInfoFrom(m))
		}
	}
	return out, nil
}

// extractNewsletters menggali []*NewsletterMetadata dari respons mex apa pun
// bentuk pembungkusnya (langsung list, {result:[...]}, {newsletters:[...]}, dst).
func extractNewsletters(data json.RawMessage) []*types.NewsletterMetadata {
	var top map[string]json.RawMessage
	if json.Unmarshal(data, &top) != nil {
		return nil
	}
	for _, raw := range top {
		// coba langsung array
		var arr []*types.NewsletterMetadata
		if json.Unmarshal(raw, &arr) == nil && len(arr) > 0 {
			return arr
		}
		// coba pembungkus {result/newsletters/edges: [...]}
		var wrap struct {
			Result      []*types.NewsletterMetadata `json:"result"`
			Newsletters []*types.NewsletterMetadata `json:"newsletters"`
		}
		if json.Unmarshal(raw, &wrap) == nil {
			if len(wrap.Result) > 0 {
				return wrap.Result
			}
			if len(wrap.Newsletters) > 0 {
				return wrap.Newsletters
			}
		}
	}
	return nil
}

// FollowChannelByJID mengikuti saluran berdasarkan JID (dari hasil rekomendasi).
func (e *Engine) FollowChannelByJID(ctx context.Context, jid string) error {
	j, err := types.ParseJID(jid)
	if err != nil {
		return err
	}
	return e.Client.FollowNewsletter(ctx, j)
}

// ChannelMessages mengambil hingga `count` pesan terbaru sebuah saluran.
func (e *Engine) ChannelMessages(ctx context.Context, jid string, count int) ([]ChannelMsg, error) {
	cj, err := types.ParseJID(jid)
	if err != nil {
		return nil, err
	}
	msgs, err := e.Client.GetNewsletterMessages(ctx, cj, &whatsmeow.GetNewsletterMessagesParams{Count: count})
	if err != nil {
		return nil, err
	}
	out := make([]ChannelMsg, 0, len(msgs))
	for _, nm := range msgs {
		kind, txt, thumb, _ := describeMessage(nm.Message)
		if kind == "" {
			continue
		}
		out = append(out, ChannelMsg{
			ID: string(nm.MessageID), ServerID: int64(nm.MessageServerID),
			Text: txt, Kind: kind, Thumb: thumb,
			Timestamp: nm.Timestamp, Views: nm.ViewsCount,
		})
	}
	return out, nil
}

// ReactChannel mengirim reaksi emoji pada satu post saluran (emoji ""=lepas).
func (e *Engine) ReactChannel(ctx context.Context, channelJID, msgID string, serverID int64, emoji string) error {
	cj, err := types.ParseJID(channelJID)
	if err != nil {
		return err
	}
	return e.Client.NewsletterSendReaction(ctx, cj, types.MessageServerID(serverID), emoji, types.MessageID(msgID))
}

// ChannelInfoByInvite mengintip metadata saluran dari tautan/kunci undangan.
func (e *Engine) ChannelInfoByInvite(ctx context.Context, link string) (ChannelInfo, error) {
	key := inviteKey(link)
	m, err := e.Client.GetNewsletterInfoWithInvite(ctx, key)
	if err != nil {
		return ChannelInfo{}, err
	}
	return channelInfoFrom(m), nil
}

// FollowChannel mengikuti saluran via tautan undangan (atau JID langsung).
func (e *Engine) FollowChannel(ctx context.Context, link string) (ChannelInfo, error) {
	var cj types.JID
	if strings.Contains(link, "@newsletter") {
		j, err := types.ParseJID(link)
		if err != nil {
			return ChannelInfo{}, err
		}
		cj = j
	} else {
		m, err := e.Client.GetNewsletterInfoWithInvite(ctx, inviteKey(link))
		if err != nil {
			return ChannelInfo{}, err
		}
		cj = m.ID
	}
	if err := e.Client.FollowNewsletter(ctx, cj); err != nil {
		return ChannelInfo{}, err
	}
	m, err := e.Client.GetNewsletterInfo(ctx, cj)
	if err != nil {
		return ChannelInfo{JID: cj.String()}, nil
	}
	return channelInfoFrom(m), nil
}

// UnfollowChannel berhenti mengikuti saluran.
func (e *Engine) UnfollowChannel(ctx context.Context, jid string) error {
	cj, err := types.ParseJID(jid)
	if err != nil {
		return err
	}
	return e.Client.UnfollowNewsletter(ctx, cj)
}

// MuteChannel membisukan / membunyikan notifikasi saluran.
func (e *Engine) MuteChannel(ctx context.Context, jid string, mute bool) error {
	cj, err := types.ParseJID(jid)
	if err != nil {
		return err
	}
	return e.Client.NewsletterToggleMute(ctx, cj, mute)
}

// --- helper ---

func channelInfoFrom(m *types.NewsletterMetadata) ChannelInfo {
	tm := m.ThreadMeta
	ci := ChannelInfo{
		JID:         m.ID.String(),
		Name:        tm.Name.Text,
		Description: tm.Description.Text,
		Subscribers: tm.SubscriberCount,
		Verified:    tm.VerificationState == types.NewsletterVerificationStateVerified,
		Invite:      tm.InviteCode,
	}
	if tm.Picture != nil && tm.Picture.URL != "" {
		ci.Picture = tm.Picture.URL
	} else if tm.Preview.URL != "" {
		ci.Picture = tm.Preview.URL
	}
	if m.ViewerMeta != nil {
		ci.Muted = m.ViewerMeta.Mute == types.NewsletterMuteOn
		ci.Role = string(m.ViewerMeta.Role)
	}
	return ci
}

// inviteKey memetik kunci dari tautan saluran (…/channel/<key>) atau kunci polos.
func inviteKey(link string) string {
	link = strings.TrimSpace(link)
	if i := strings.LastIndex(link, "/channel/"); i >= 0 {
		link = link[i+len("/channel/"):]
	} else if i := strings.LastIndexByte(link, '/'); i >= 0 {
		link = link[i+1:]
	}
	if i := strings.IndexByte(link, '?'); i >= 0 {
		link = link[:i]
	}
	return link
}
