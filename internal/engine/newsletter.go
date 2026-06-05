package engine

// newsletter.go — WhatsApp Channels (newsletter). Daftar saluran yang diikuti,
// ambil pesannya (read-only), ikuti/berhenti via tautan undangan.

import (
	"context"
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
