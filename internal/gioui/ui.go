// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 Muhammad Mishbakhuz Zuhail
//
// ui.go — tata letak 3-panel (rail | sidebar | percakapan), daftar chat, bubble
// pesan, avatar. Menggambar bentuk kustom (RRect bubble, lingkaran avatar) via
// clip — membuktikan Gio bisa desain pixel-WhatsApp. Data dari engine in-process.
package gioui

import (
	"encoding/base64"
	"encoding/json"
	"image"
	"image/color"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/io/clipboard"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/muhammadmishbakhuzzuhail/whatslite/internal/app"
)

type UI struct {
	th   *material.Theme
	core *app.App
	t    Theme
	dark bool

	// kunci aplikasi (PIN): gate sebelum UI utama + dialog atur/hapus di setelan.
	locked      bool
	pinEd       widget.Editor // input PIN di lock-screen
	pinErr      bool
	pinSetEd    widget.Editor // input PIN baru (dialog atur)
	pinSetBtn   widget.Clickable
	pinClearBtn widget.Clickable
	pinCancel   widget.Clickable

	state      string
	qr         string // kode QR pairing mentah terbaru (dari core.QRCode); "" = belum ada
	subtitle   string // subtitle header chat (online/mengetik…/terakhir dilihat)
	groupSub   string // ringkasan anggota grup terbuka (nama2 / "N anggota") utk subtitle
	typingSent bool   // status composing terakhir yg dikirim (throttle SendTyping)
	view       string // pane sidebar aktif: chats|calls|settings

	olderReqChat string    // chat terakhir diminta history lama (throttle pagination)
	olderReqAt   time.Time // waktu permintaan history lama terakhir

	pollClicks    map[string][]widget.Clickable // msgID → clickable per opsi polling
	pollVoteCache map[string]pollVoteEntry      // msgID → hasil suara (TTL — hindari query/frame)
	mentionState  rtInteractiveText             // state teks ber-mention (warna inline)
	linkStates    map[string]*rtInteractiveText // state per-pesan utk URL klik

	// picker stiker (tombol stiker composer → overlay "picker").
	stickerClick  widget.Clickable
	pickerScrim   widget.Clickable // ketuk luar kartu picker → tutup
	stickerCache  []app.StickerDTO
	stickerThumbs map[string]paint.ImageOp // hash → thumbnail
	stickerTried  map[string]bool
	stickerClicks []widget.Clickable

	statusGroupsCache []app.StatusGroupDTO // grup status terkini (utk viewer)
	statusClicks      []widget.Clickable
	statusList        widget.List // gulir daftar status (section 2)
	statusViewIdx     int
	statusItemIdx     int       // item ke-berapa dlm grup yg sedang dilihat (tap-through)
	statusViewAt      time.Time // waktu buka viewer (redraw terbatas saat unduh media)
	statusClose       widget.Clickable
	stPrevZone        widget.Clickable // zona tap kiri → item sebelumnya
	stNextZone        widget.Clickable // zona tap kanan → item berikut / tutup
	stReplyEd         widget.Editor    // balas status (kirim DM ke poster)
	stReplySend       widget.Clickable
	stEmoji           [6]widget.Clickable // reaksi emoji cepat (ala IG story)
	stEmojiMore       widget.Clickable    // tombol "+" → buka picker emoji lengkap
	stReplied         string              // emoji terkirim terakhir (umpan balik singkat)
	stPaused          bool                // jeda auto-advance (tombol pause)
	stPause           widget.Clickable    // toggle pause/main
	stItemStart       time.Time           // waktu item kini mulai (utk progress auto-advance)
	stVid             StatusVideo         // sesi video status inline (frame+audio); nil = tak ada
	stVidID           string              // id item video yg sedang dimuat
	stFwd             widget.Clickable    // tombol forward status → chat
	fwdSrc            string              // chat sumber forward ("" = u.selected; status → status@broadcast)
	stMyClick         widget.Clickable    // baris "Status saya" → composer post status
	scEd              widget.Editor       // editor teks status (composer post)
	scPost            widget.Clickable
	scCancel          widget.Clickable
	scMedia           widget.Clickable // composer: pilih foto/video → status media

	contactFlat       []app.ContactRowDTO // kontak datar (pane Kontak → buka chat)
	contactPaneClicks []widget.Clickable

	// buat grup (Kontak → "Grup baru"): nama + multi-pilih kontak.
	gcNewBtn   widget.Clickable
	gcNameEd   widget.Editor
	gcSel      map[string]bool // jid → terpilih
	gcClicks   []widget.Clickable
	gcCreate   widget.Clickable
	gcMode     string // ""/"create" = buat grup; "addmember" = tambah anggota ke gcGroupJID
	gcGroupJID string // grup sasaran saat gcMode == "addmember"
	gcCancel   widget.Clickable
	gcList     widget.List

	// cache TTL pembangun data pane (hindari query DB tiap frame saat scroll pane).
	cgCache                       []cpGroup
	srCache                       []stpItem
	crCache                       []spCall
	chCache                       []chnChannel
	comCache                      []comItem
	cgAt, srAt, crAt, chAt, comAt time.Time

	chnTab       int                 // 0=Diikuti, 1=Jelajahi
	chnTabClicks [2]widget.Clickable // tombol tab channels
	chnRowClicks []widget.Clickable  // aksi per-baris channel (ikuti/unfollow)
	chnRowOpens  []widget.Clickable  // buka channel per-baris → reader
	openChannel  string              // jid channel terbuka (reader di section 3); "" = tak ada
	openChanName string              // nama channel terbuka
	openChanSubs string              // subscriber channel terbuka (header reader)
	openChanVer  bool                // channel terverifikasi (header reader)
	chMsgList    widget.List         // gulir post channel reader
	chMsgsCache  []app.ChannelMsgDTO // cache post channel terbuka
	chMsgsJID    string              // jid cache di atas
	chMsgsAt     time.Time
	chnExpCache  []chnChannel // cache saluran jelajah
	chnExpAt     time.Time
	chnExpQuery  string        // query terakhir direktori jelajah (invalidasi cache)
	chnSearchEd  widget.Editor // kotak cari direktori channels (tab Jelajahi)
	ctSearchEd   widget.Editor // kotak cari daftar Kontak (section 2)
	cgQuery      string        // query terakhir daftar kontak (invalidasi cache)

	// alur login via nomor telepon (alternatif QR): toggle, input, kode 8-karakter.
	loginPhone  bool
	phoneEd     widget.Editor
	loginSwitch widget.Clickable
	loginSubmit widget.Clickable
	pairCode    string

	setClicks  [11]widget.Clickable // baris pane setelan (lihat setList: 0=Akun … 9=Bantuan, 10=Keluar)
	langClicks [8]widget.Clickable  // baris pemilih bahasa (sub-pane Bahasa)

	// pencarian + filter daftar chat (paritas SearchBar.svelte + Filters.svelte).
	searchEd     widget.Editor
	filterSel    int // 0 Semua · 1 Belum dibaca · 2 Favorit · 3 Grup
	filterClicks [4]widget.Clickable
	shown        []int            // indeks u.chats yg lolos filter+pencarian (urut tampil)
	newChatClick widget.Clickable // baris "mulai chat baru" (query nomor)

	// pencarian pesan global (ikon cari header → view "search").
	svEd           widget.Editor
	svHits         []svHit
	svHitClicks    []widget.Clickable
	svBack         widget.Clickable
	svPrevView     string           // view sebelum pencarian (utk kembali)
	searchMsgClick widget.Clickable // baris "Cari pesan '<q>'" → masuk view search

	// panel pesan berbintang (chat-overflow → "Pesan berbintang", view "starred").
	starHits      []svHit
	starHitClicks []widget.Clickable
	starBack      widget.Clickable
	starAt        time.Time // TTL cache GetStarred

	// mode balas: pesan yg dikutip; "" = kirim biasa.
	replyTo   string
	replyName string
	replyText string
	replyX    widget.Clickable // tombol batal balas

	editTarget string           // msgID yg sedang diedit ("" = kirim biasa)
	editText   string           // teks asli (banner edit)
	editCancel widget.Clickable // tombol batal edit

	pinnedCache []app.MessageDTO // pesan tersemat chat aktif (GetPinned, TTL 2s)
	pinnedAt    time.Time
	pinnedChat  string           // chat yg pinnedCache-nya valid
	pinnedBar   widget.Clickable // bar pesan-tersemat → lompat ke pesan
	pinnedIdx   int              // pin yg sedang ditampilkan (ketuk = siklus berikutnya)

	drafts    map[string]string // draft composer per-chat (jid → teks belum terkirim)
	draftChat string            // chat yg draft-nya sedang ada di editor

	linkMu    sync.Mutex
	linkPrev  map[string]*app.LinkPreviewDTO // pratinjau tautan per-URL (async)
	linkImg   map[string]paint.ImageOp       // thumbnail pratinjau (decode async)
	linkTried map[string]bool                // URL sudah dicoba ambil

	transMu    sync.Mutex
	transText  map[string]string // msgID → teks terjemahan (async)
	transTried map[string]bool   // msgID sudah dicoba terjemah
	expanded   map[string]bool   // msgID → teks panjang dibentangkan (show more/less)
	moreClicks map[string]*widget.Clickable

	// pratinjau media sebelum kirim (pilih berkas → overlay caption + sekali-lihat).
	pendMu      sync.Mutex
	pendHas     bool
	pendKind    string // image | video | document
	pendURI     string // data-URI berkas terpilih
	pendImg     paint.ImageOp
	pendImgHas  bool
	capEd       widget.Editor    // caption media
	pendVO      bool             // sekali-lihat (view-once)
	pendVOClick widget.Clickable // toggle sekali-lihat
	pendSend    widget.Clickable
	pendCancel  widget.Clickable
	pendRotate  widget.Clickable // putar 90° (image)
	pendCrop    widget.Clickable // terapkan potong
	cropActive  bool             // ada seleksi potong
	cropA       image.Point      // sudut awal seleksi (koord box gambar)
	cropB       image.Point      // sudut akhir seleksi
	cropTagV    int              // tag pointer area potong

	unreadDivID    string // ID pesan tempat divider "belum dibaca" digambar ("" = tak ada)
	unreadDivCount int    // jumlah belum-dibaca saat chat dibuka

	// pemilih reaksi: target pesan (kosong = mode sisip emoji ke editor).
	rpClicks    []widget.Clickable
	rpList      widget.List
	reactMsgID  string
	reactSender string
	reactFromMe bool

	// teruskan: id pesan sumber + klik per-chat tujuan + batal.
	fwdMsgID  string
	fwdMsgIDs []string // teruskan banyak (mode pilih); kosong = pakai fwdMsgID
	fwdClicks []widget.Clickable
	fwdCancel widget.Clickable
	fwdSel    map[string]bool // tujuan forward terpilih (multi-target)
	fwdSend   widget.Clickable

	// mode pilih (multi-select) → aksi massal hapus/teruskan.
	selMode   bool
	selSet    map[string]bool // msgID terpilih
	selCancel widget.Clickable
	selDelete widget.Clickable
	selFwd    widget.Clickable

	// jadwalkan pesan: 3 preset waktu + batal (ScheduleMessage).
	schedItems  [3]widget.Clickable
	schedCancel widget.Clickable

	// menu lampiran (tombol "+"): klik per-baris.
	attachClicks []widget.Clickable

	// penyusun polling (lampiran → Polling): pertanyaan + opsi + tombol.
	pollQEd    widget.Editor
	pollOptEds [4]widget.Editor
	pollCreate widget.Clickable
	pollCancel widget.Clickable

	// kirim kontak (lampiran → Kontak): daftar kontak + pilih.
	contactSendCache  []app.ContactRowDTO
	contactSendClicks []widget.Clickable
	contactSendCancel widget.Clickable
	contactSendList   widget.List

	// kirim lokasi (lampiran → Lokasi): nama tempat + lat/lng.
	locNameEd widget.Editor
	locLatEd  widget.Editor
	locLngEd  widget.Editor
	locSend   widget.Clickable
	locCancel widget.Clickable

	// menu aksi baris chat (klik-kanan): SNAPSHOT chat saat menu dibuka (aksi pakai
	// ini, bukan index — u.chats di-replace tiap refresh & bisa reorder).
	chatCtxChat      app.ChatDTO
	chatCtxItems     [6]widget.Clickable
	headMenuClick    widget.Clickable // ikon overflow header → menu chat terbuka
	headSearchClick  widget.Clickable // ikon cari header → cari DALAM chat aktif
	infoBlockC       widget.Clickable // info-drawer: blokir kontak
	infoLeaveC       widget.Clickable // info-drawer: keluar grup
	infoInviteC      widget.Clickable // info-drawer: link undangan grup
	inviteLink       string           // link undangan termuat (modal "invitelink")
	inviteCopy       widget.Clickable
	inviteClose      widget.Clickable
	infoEditC        widget.Clickable // info-drawer: edit info grup
	infoRenameC      widget.Clickable // info-drawer: edit nama kontak (DM)
	renameEd         widget.Editor    // editor nama kontak (modal renamecontact)
	renameSave       widget.Clickable
	renameCancel     widget.Clickable
	delTarget        app.MessageDTO // pesan yg akan dihapus (dialog hapus untuk saya/semua)
	delForMe         widget.Clickable
	delForAll        widget.Clickable
	delCancel        widget.Clickable
	renameTarget     string             // jid sasaran rename (kontak peek / ctx); "" = pakai selected
	infoCJID         string             // "intip-info" kontak (drawer TANPA buka chat); "" = pakai selected
	infoCName        string             // nama kontak utk intip-info
	infoClicks       []widget.Clickable // pane Kontak: ikon "i" per-baris → info-drawer
	ctNewContactBtn  widget.Clickable   // pane Kontak: tombol "Kontak baru"
	ncName           widget.Editor      // modal newcontact: nama
	ncPhone          widget.Editor      // modal newcontact: nomor
	ncSave           widget.Clickable
	ncCancel         widget.Clickable
	ncErr            string                  // pesan galat modal newcontact (nomor tak terdaftar)
	cctContact       app.ContactRowDTO       // snapshot kontak utk menu konteks (klik-kanan)
	cctMsg           widget.Clickable        // menu konteks kontak: kirim pesan
	cctInfo          widget.Clickable        // menu konteks kontak: info kontak
	cctRename        widget.Clickable        // menu konteks kontak: edit nama
	cctBlock         widget.Clickable        // menu konteks kontak: blokir/buka blokir
	cctDelete        widget.Clickable        // menu konteks kontak: hapus kontak
	infoMuteC        widget.Clickable        // info-drawer: bisukan/aktifkan notifikasi
	infoMediaC       widget.Clickable        // info-drawer: buka galeri media
	infoEncC         widget.Clickable        // info-drawer: info enkripsi
	infoMemberClicks []widget.Clickable      // info-drawer: anggota grup
	infoMemberJIDs   []string                // jid anggota (paralel infoMemberClicks)
	commonGroups     map[string][]InfoMember // jid kontak → grup bersama (cache async)
	commonTried      map[string]time.Time    // cooldown fetch grup bersama
	commonMu         sync.Mutex
	commonClicks     []widget.Clickable // paralel grup-bersama DM terbuka (tap → buka grup)
	// gabung grup lewat tautan undangan:
	joinLink        string              // tautan/kode yg sedang dipratinjau
	joinPreview     string              // nama grup hasil PreviewGroupLink (async)
	joinClick       widget.Clickable    // baris "Gabung grup lewat tautan" di daftar chat
	joinConfirm     widget.Clickable    // tombol Gabung di modal
	joinCancel      widget.Clickable    // tombol Batal di modal
	infoMemberNames []string            // nama anggota (paralel)
	infoMemberAdmin []bool              // status admin anggota (paralel)
	infoAddC        widget.Clickable    // info-drawer: "Tambah anggota" (grup)
	infoAnnounceC   widget.Clickable    // toggle: hanya admin boleh kirim
	infoLockedC     widget.Clickable    // toggle: hanya admin boleh ubah info
	infoApprovalC   widget.Clickable    // toggle: setujui anggota baru
	curGroupAmAdmin bool                // saya admin grup terbuka? (gate aksi admin)
	mctJID          string              // menu konteks anggota: jid sasaran
	mctName         string              // nama sasaran
	mctAdmin        bool                // sasaran sudah admin?
	mctMsg          widget.Clickable    // "Kirim pesan"
	mctPromote      widget.Clickable    // "Jadikan admin" / "Hapus admin"
	mctRemove       widget.Clickable    // "Keluarkan dari grup"
	mctClose        widget.Clickable    // "Tutup"
	encClose        widget.Clickable    // overlay enkripsi/galeri: tutup
	mediaCellClicks []widget.Clickable  // sel grid galeri media
	mediaGalList    widget.List         // scroll galeri media
	infoTimerC      widget.Clickable    // info-drawer: pesan sementara (buka picker)
	dispClicks      [4]widget.Clickable // picker pesan sementara: Mati/24j/7h/90h
	dispClose       widget.Clickable    // picker pesan sementara: tutup
	dispTimer       map[string]int      // jid → timer detik terpilih (label drawer)
	gedName         widget.Editor       // editor nama grup (modal groupedit)
	gedDesc         widget.Editor       // editor deskripsi grup
	gedSave         widget.Clickable
	gedCancel       widget.Clickable
	curGroupDesc    string // deskripsi grup aktif (utk prefill editor)

	inChatSearch bool          // mode cari-dalam-chat aktif (header → bilah cari)
	inChatEd     widget.Editor // input cari-dalam-chat
	inChatBack   widget.Clickable
	inChatPrev   widget.Clickable
	inChatNext   widget.Clickable
	inChatMatch  []int // indeks u.messages yg cocok query
	inChatCur    int   // match aktif (0-based)

	// sub-pane setelan (profil/penyimpanan) + navigasi kembali.
	setSub          string
	setBack         widget.Clickable
	setProfileClick widget.Clickable
	profNameEd      widget.Editor // edit nama (sub-pane profil)
	profAboutEd     widget.Editor // edit tentang
	profSave        widget.Clickable
	profLoaded      bool                // editor sudah diisi nilai saat ini?
	profName        string              // nama profil sendiri (avatar rail)
	selfJID         string              // JID sendiri (foto profil avatar rail)
	profFetched     bool                // profil sudah diambil sekali
	privacyClicks   [8]widget.Clickable // baris privasi (siklus nilai → SetPrivacy)

	chats     []app.ChatDTO
	selected  string
	selName   string
	selGroup  bool
	messages  []app.MessageDTO
	lastFetch time.Time

	chatList         widget.List
	msgList          widget.List
	contactList      widget.List
	clicks           []widget.Clickable
	railClicks       []widget.Clickable
	railProfileClick widget.Clickable     // avatar profil di dasar rail → setelan profil
	profPhotoClick   widget.Clickable     // avatar di pane profil → ganti foto
	composeMentions  map[string]string    // number → jid yg di-mention di pesan sekarang
	memCacheJID      string               // grup yg memCache valid
	memCache         []app.GroupMemberDTO // anggota grup (cache utk saran @mention)
	memAt            time.Time
	mentionClicks    []widget.Clickable   // klik saran @mention
	comNewBtn        widget.Clickable     // tombol "Komunitas baru" di pane Komunitas
	comOpen          string               // jid komunitas yg dibuka (detail sub-grup); "" = daftar
	comRowClicks     []widget.Clickable   // paralel komunitas (tap → buka detail)
	comSubClicks     []widget.Clickable   // paralel sub-grup komunitas terbuka (tap → buka chat)
	comBack          widget.Clickable     // tombol ← di detail komunitas
	railMetaC        widget.Clickable     // tombol Meta AI di rail (section 1)
	aboutToggle      widget.Clickable     // chevron buka/tutup dropdown saran Tentang
	aboutOpen        bool                 // dropdown saran Tentang terbuka?
	demoTypingJID    string               // render-tool: paksa indikator mengetik utk jid ini
	demoTypingWho    string               // render-tool: nama pengetik grup (avatar bubble)
	demoTypingWhoJID string               // render-tool: jid pengetik grup
	aboutClicks      [11]widget.Clickable // chip saran "Tentang" (profil)
	editor           widget.Editor
	photos           map[string]paint.ImageOp // foto avatar in-memory (nama → op)
	photoMu          sync.Mutex               // lindungi photos (diisi dari goroutine loader)
	photoTried       map[string]time.Time     // jid → waktu coba terakhir (retry stlh cooldown bila gagal)

	media      map[string]paint.ImageOp // thumbnail media bubble (msgID → op)
	mediaMu    sync.Mutex
	mediaTried map[string]bool // msgID yg sudah dicoba ambil

	overlay     string // popup aktif: ""|info|reaction|forward|msginfo|picker|lightbox|msgctx
	headerClick widget.Clickable
	emojiClick  widget.Clickable
	sendClick   widget.Clickable // tombol kirim (muncul saat ada teks; ganti mic)
	micClick    widget.Clickable // tombol mic → mulai rekam voice note
	recDemo     bool             // render-tool: paksa bar rekam
	recCancel   widget.Clickable // batal rekam
	recSend     widget.Clickable // kirim rekaman
	attachClick widget.Clickable
	backdrop    widget.Clickable
	msgClicks   []widget.Clickable
	fabClick    widget.Clickable   // tombol bulat gulir-ke-bawah (tampil saat tergulir naik)
	quoteClicks []widget.Clickable // ketuk kutipan balasan → lompat ke pesan asal
	hlMsg       string             // pesan yg sedang disorot (lompatan kutipan)
	hlAt        time.Time          // waktu mulai sorot (pudar ~1.6s)
	ctxIdx      int                // index pesan utk context-menu (display only)
	ctxMsg      app.MessageDTO     // SNAPSHOT pesan saat menu dibuka — aksi pakai ini, bukan
	// index: backfill history prepend & refresh reorder menggeser semua index.
	ctxItems [12]widget.Clickable // item menu (base6 + salin + edit/unduh + pin + pilih + terjemah)

	lightboxMsg   string           // msgID gambar yg dibuka di lightbox ("" = tutup)
	lightboxCap   string           // caption gambar lightbox
	lightboxClose widget.Clickable // tombol ✕ tutup lightbox
	lightboxSave  widget.Clickable // tombol unduh → simpan media ke disk

	// OnPlayVoice/OnPlayVideo: hook media (di-set cmd/whatslite-gio → internal/
	// voice + internal/video). gioui TETAP bebas-cgo (gio-shot ringan).
	OnPlayVoice func(chat, id string)
	OnPlayVideo func(chat, id, typ string)
	// OnAttach: hook pilih-berkas + kirim (di-set cmd/whatslite-gio → x/explorer +
	// core.SendMedia). category ∈ media|document|contact|location|poll. Pisah dari
	// gioui agar tetap bebas-window/cgo.
	OnAttach func(chat, category string)
	// OnSaveMedia: hook simpan media ke disk (di-set cmd/whatslite-gio → x/explorer
	// CreateFile + tulis MediaBytes). name = nama berkas saran.
	OnSaveMedia func(chat, id, name string)
	// OnStatusMedia: hook pilih foto/video + unggah sbg status sendiri (di-set
	// cmd/whatslite-gio → x/explorer + core.PostMediaStatus).
	OnStatusMedia func()
	// OnStatusVideo: buka video status utk putar INLINE (frame ffmpeg + audio mpv),
	// di-set cmd/whatslite-gio → internal/video. nil → video pakai thumbnail saja.
	OnStatusVideo func(id string) StatusVideo
	// OnOpenURL: buka URL di browser (di-set cmd/whatslite-gio → xdg-open). nil → tak diklik.
	OnOpenURL func(url string)
	// OnSetPhoto: pilih berkas gambar → SetMyPhoto (di-set cmd/whatslite-gio → x/explorer).
	OnSetPhoto func()
	// OnMediaPoster: dekode frame still (poster) dari byte yg tak bisa image.Decode —
	// GIF WhatsApp (mp4), stiker webp animasi, video — via ffmpeg. ext mis. ".mp4"/
	// ".webp". di-set cmd/whatslite-gio → internal/video.FirstFrame. nil → fallback ikon.
	OnMediaPoster func(data []byte, ext string) image.Image
	// OnWinAction: hook aksi window utk titlebar custom (CSD Wayland). action ∈
	// minimize|maximize|unmaximize|close. nil (gio-shot) → titlebar statis.
	OnWinAction func(action string)

	winMin   widget.Clickable // tombol minimize titlebar
	winMax   widget.Clickable // tombol maximize/restore titlebar
	winClose widget.Clickable // tombol close titlebar
	winMaxed bool             // status maximize (toggle ikon + aksi)
}

// ctxMenu = item context-menu pesan (glyph + aksi/overlay tujuan).
var ctxMenu = []struct{ icon, label, to string }{
	{"emoji", "Reaksi", "reaction"}, {"reply", "Balas", ""}, {"forward", "Teruskan", "forward"},
	{"star", "Bintangi", ""}, {"info", "Info", "msginfo"}, {"trash", "Hapus", "delconfirm"},
}

// SetOverlay: utk render-tool menguji popup headless.
func (u *UI) SetOverlay(o string) { u.overlay = o }

// SetDeleteConfirmDemo: render-tool — dialog hapus pesan (own=true → 3 tombol).
func (u *UI) SetDeleteConfirmDemo() {
	u.delTarget = app.MessageDTO{ID: "m2", Dir: "out", Type: "text", Text: "halo"}
	u.overlay = "delconfirm"
}

// SetCommunityDemo: render-tool — detail komunitas (daftar sub-grup) headless.
func (u *UI) SetCommunityDemo() {
	u.view = "communities"
	u.comCache = []comItem{
		{jid: "c1@g.us", name: "Tim Kantor", sub: "3 grup", groups: []comSub{
			{jid: "g0@g.us", name: "Pengumuman", isDefault: true},
			{jid: "g1@g.us", name: "Umum"},
			{jid: "g2@g.us", name: "Proyek X"},
		}},
	}
	u.comAt = u.comAt.Add(0) // jaga cache valid (tak penting di demo)
	u.comOpen = "c1@g.us"
}

// SetForwardDemo: render-tool — modal teruskan dgn 2 tujuan terpilih (multi-target).
func (u *UI) SetForwardDemo() {
	u.fwdMsgID = "m1"
	u.overlay = "forward"
	for i, c := range u.chats {
		if i == 0 || i == 2 {
			u.fwdSel[c.ID] = true
		}
	}
}

// SetMentionDemo: render-tool — popup saran @mention di grup (ketik "@").
func (u *UI) SetMentionDemo() {
	u.selGroup, u.selected = true, "g1@g.us"
	u.memCache = []app.GroupMemberDTO{
		{JID: "6281111@s.whatsapp.net", Name: "Andi Pratama"},
		{JID: "6282222@s.whatsapp.net", Name: "Sarah"},
		{JID: "6283333@s.whatsapp.net", Name: "Budi Santoso"},
	}
	u.editor.SetText("Halo @")
}

// SetMsgCtxDemo: render-tool — dropdown aksi pesan teks (incl. Terjemahkan).
func (u *UI) SetMsgCtxDemo() {
	u.ctxMsg = app.MessageDTO{ID: "m1", Dir: "in", Type: "text", Text: "Hello everyone", Sender: "Budi"}
	u.overlay = "msgctx"
}

// SetJoinLinkDemo: render-tool — modal gabung grup lewat tautan (pratinjau termuat).
func (u *UI) SetJoinLinkDemo() {
	u.joinLink, u.joinPreview, u.overlay = "https://chat.whatsapp.com/ABC123", "Grup Kerja", "joinlink"
}

// SetMemberCtxDemo: render-tool — menu konteks anggota grup (admin → semua aksi).
func (u *UI) SetMemberCtxDemo() {
	u.mctJID, u.mctName, u.mctAdmin = "62811@s.whatsapp.net", "Sarah", false
	u.curGroupAmAdmin = true
	u.overlay = "memberctx"
}

// SetLightbox: utk render-tool membuka lightbox gambar nyata headless.
func (u *UI) SetLightbox(id, cap string) {
	u.lightboxMsg, u.lightboxCap, u.overlay = id, cap, "lightbox"
}

// SetHighlight: utk render-tool menyorot pesan (lompatan kutipan) headless.
func (u *UI) SetHighlight(id string) { u.hlMsg, u.hlAt = id, time.Now() }

// SetComposeText: utk render-tool mengisi composer (uji tombol kirim) headless.
func (u *UI) SetComposeText(s string) { u.editor.SetText(s) }

// SetEditing: utk render-tool menguji banner edit headless.
func (u *UI) SetEditing(id, text string) {
	u.editTarget, u.editText = id, text
	u.editor.SetText(text)
}

// SetTranslateDemo: utk render-tool menguji baris terjemahan headless.
func (u *UI) SetTranslateDemo(id, text string) { u.transText[id] = text }

// SetMediaPreviewDemo: utk render-tool menguji pratinjau media headless.
func (u *UI) SetMediaPreviewDemo() {
	u.pendKind, u.pendImg, u.pendImgHas, u.pendHas = "image", synthPhoto(), true, true
	u.capEd.SetText("Foto liburan kemarin")
	u.overlay = "mediapreview"
}

// SetPendingMedia — dipanggil cmd setelah pilih berkas (media): simpan untuk
// pratinjau (caption + sekali-lihat) sebelum kirim. Aman dari goroutine (mutex);
// overlay dibuka di Layout (thread UI). kind: image|video|document.
func (u *UI) SetPendingMedia(kind, dataURI string) {
	var op paint.ImageOp
	hasImg := false
	if kind == "image" {
		if img := decodeImage(decodeDataURI(dataURI)); img != nil {
			op, hasImg = paint.NewImageOp(img), true
		}
	}
	u.pendMu.Lock()
	u.pendKind, u.pendURI, u.pendImg, u.pendImgHas = kind, dataURI, op, hasImg
	u.pendHas, u.pendVO = true, false
	u.pendMu.Unlock()
	u.capEd.SetText("")
}

func (u *UI) clearPending() {
	u.pendMu.Lock()
	u.pendHas, u.pendURI, u.pendImgHas, u.pendVO = false, "", false, false
	u.pendMu.Unlock()
	u.cropActive = false
	u.capEd.SetText("")
}

// SetRecordingDemo: utk render-tool menguji bar rekam voice note headless.
func (u *UI) SetRecordingDemo() { u.recDemo = true }

// SetGroupEditDemo: utk render-tool menguji modal edit info grup headless.
func (u *UI) SetGroupEditDemo(name, desc string) {
	u.gedName.SetText(name)
	u.gedDesc.SetText(desc)
	u.overlay = "groupedit"
}

// SetInviteDemo: utk render-tool menguji modal link undangan headless.
func (u *UI) SetInviteDemo(link string) { u.inviteLink = link; u.overlay = "invitelink" }

// SetDisappearingDemo: render-tool — pilih chat + buka picker pesan sementara.
func (u *UI) SetDisappearingDemo() {
	if len(u.chats) > 0 {
		u.selected = u.chats[0].ID
	}
	u.overlay = "disappearing"
}

// SetLinkPreviewDemo: utk render-tool menguji kartu pratinjau tautan headless
// (inject preview + thumbnail sintetis, gulir ke bawah agar terlihat).
func (u *UI) SetLinkPreviewDemo(url, title, desc string) {
	site, video := "", false
	switch h := strings.ToLower(url); {
	case strings.Contains(h, "tiktok"):
		site, video = "TikTok", true
	case strings.Contains(h, "youtu"):
		site, video = "YouTube", true
	case strings.Contains(h, "instagram"):
		site, video = "Instagram", true
	}
	u.linkPrev[url] = &app.LinkPreviewDTO{URL: url, Title: title, Desc: desc, Image: "x", Site: site, Video: video}
	u.linkImg[url] = synthPhoto()
	u.linkTried[url] = true
	u.msgList.ScrollTo(9999)
}

// SetSelectDemo: utk render-tool menguji mode-pilih (toolbar + sorot) headless.
func (u *UI) SetSelectDemo(ids ...string) {
	u.selMode = true
	for _, id := range ids {
		u.selSet[id] = true
	}
}

// SetInChatSearch: utk render-tool menguji bilah cari-dalam-chat headless.
func (u *UI) SetInChatSearch(q string) {
	u.inChatSearch = true
	u.inChatEd.SetText(q)
}

// SetUnreadDemo: utk render-tool menguji divider "belum dibaca" headless.
func (u *UI) SetUnreadDemo(id string, n int) { u.unreadDivID, u.unreadDivCount = id, n }

// SetPinnedDemo: utk render-tool menguji bar pesan-tersemat headless.
func (u *UI) SetPinnedDemo(text string, n int) {
	u.pinnedCache = make([]app.MessageDTO, n)
	for i := range u.pinnedCache {
		u.pinnedCache[i] = app.MessageDTO{ID: "m3", Type: "text", Text: text}
	}
	u.pinnedChat, u.pinnedAt = u.selected, time.Now().Add(time.Hour) // jaga cache valid
}

// SetReply: utk render-tool menguji banner balas headless.
func (u *UI) SetReply(name, text string) { u.replyTo, u.replyName, u.replyText = "demo", name, text }

// ScrollMessagesToEnd: utk render-tool menguji gulir-ke-bawah headless.
func (u *UI) ScrollMessagesToEnd() { u.msgList.ScrollTo(1 << 20) }

// SetSearch: utk render-tool menguji bilah cari / tawaran chat-baru headless.
func (u *UI) SetSearch(s string) { u.searchEd.SetText(s) }

// SetLocked: utk render-tool menguji lock-screen headless.
func (u *UI) SetLocked(b bool) { u.locked = b }

// SetSettingsSub: utk render-tool menguji sub-pane setelan headless.
func (u *UI) SetSettingsSub(s string) { u.view = "settings"; u.setSub = s }

// railNav = tombol nav rail kiri (ikon SVG WhatsApp + view tujuan).
var railNav = []struct{ view, icon, label string }{
	{"chats", "chats", "Chat"}, {"status", "status", "Status"}, {"channels", "channels", "Saluran"},
	{"communities", "communities", "Komunitas"}, {"calls", "calls", "Panggilan"}, {"contacts", "contacts", "Kontak"},
	{"settings", "settings", "Setelan"},
}

func NewUI(th *material.Theme, core *app.App) *UI {
	u := &UI{th: th, core: core, dark: true, view: "chats"}
	if core != nil {
		u.dark = core.ThemeDark() // pulihkan preferensi tema tersimpan
	}
	u.t = newTheme(u.dark)
	u.chatList.Axis = layout.Vertical
	u.msgList.Axis = layout.Vertical
	u.contactList.Axis = layout.Vertical
	u.statusList.Axis = layout.Vertical
	u.mediaGalList.Axis = layout.Vertical
	u.chnSearchEd.SingleLine = true
	u.renameEd.SingleLine, u.renameEd.Submit = true, true
	u.ctSearchEd.SingleLine = true
	u.ncName.SingleLine = true
	u.ncPhone.SingleLine, u.ncPhone.Submit = true, true
	u.stReplyEd.SingleLine, u.stReplyEd.Submit = true, true
	u.scEd.Submit = true
	u.railClicks = make([]widget.Clickable, len(railNav))
	u.editor.SingleLine = false // multibaris: Shift+Enter = baris baru, Enter = kirim
	u.editor.Submit = true
	u.inChatEd.SingleLine = true
	u.inChatEd.Submit = true
	u.gedName.SingleLine = true
	u.phoneEd.SingleLine = true
	u.phoneEd.Submit = true
	u.searchEd.SingleLine = true
	u.profNameEd.SingleLine = true
	u.profNameEd.MaxLen = 25 // batas nama WhatsApp
	u.profAboutEd.SingleLine = true
	u.profAboutEd.MaxLen = 139 // batas Tentang WhatsApp
	u.pollQEd.SingleLine = true
	for i := range u.pollOptEds {
		u.pollOptEds[i].SingleLine = true
	}
	u.locNameEd.SingleLine = true
	u.locLatEd.SingleLine = true
	u.locLngEd.SingleLine = true
	u.gcNameEd.SingleLine = true
	u.gcSel = map[string]bool{}
	u.svEd.SingleLine = true
	u.pinEd.SingleLine, u.pinEd.Submit, u.pinEd.Mask = true, true, '•'
	u.pinEd.Filter = "0123456789" // PIN = digit saja
	// pinSetEd: Submit=true agar Enter MENGONFIRMASI (bukan menyisip '•'/newline).
	u.pinSetEd.SingleLine, u.pinSetEd.Submit, u.pinSetEd.Mask = true, true, '•'
	u.pinSetEd.Filter = "0123456789"
	u.locked = core != nil && core.HasAppPIN()
	u.rpClicks = make([]widget.Clickable, len(RpEmoji()))
	u.photos = map[string]paint.ImageOp{}
	u.photoTried = map[string]time.Time{}
	u.media = map[string]paint.ImageOp{}
	u.mediaTried = map[string]bool{}
	u.dispTimer = map[string]int{}
	u.drafts = map[string]string{}
	u.selSet = map[string]bool{}
	u.commonGroups = map[string][]InfoMember{}
	u.commonTried = map[string]time.Time{}
	u.composeMentions = map[string]string{}
	u.fwdSel = map[string]bool{}
	u.linkPrev = map[string]*app.LinkPreviewDTO{}
	u.linkImg = map[string]paint.ImageOp{}
	u.linkTried = map[string]bool{}
	u.transText = map[string]string{}
	u.transTried = map[string]bool{}
	u.expanded = map[string]bool{}
	u.moreClicks = map[string]*widget.Clickable{}
	u.pollClicks = map[string][]widget.Clickable{}
	u.pollVoteCache = map[string]pollVoteEntry{}
	u.stickerThumbs = map[string]paint.ImageOp{}
	u.stickerTried = map[string]bool{}
	if core == nil { // demo: foto sintetis utk membuktikan avatar-foto bulat + thumb
		u.photos["Andi Pratama"] = synthPhoto()
		u.media["m13"] = synthPhoto() // bubble image demo (m14 video = placeholder+play)
	}
	return u
}

// SetDark: ganti tema (dipakai render-tool utk audit light/dark).
func (u *UI) SetDark(d bool) { u.dark = d; u.t = newTheme(d) }

// SetView/Deselect: utk render-tool menguji state navigasi headless.
func (u *UI) SetView(v string) { u.view = v }
func (u *UI) Deselect() {
	u.selected = ""
	if u.core != nil {
		u.core.CloseChat()
	}
}

// View/Overlay — getter agar render-tool bisa simpan+pulihkan state saat memotret
// layar bernama dari app yg sedang berjalan (WLGIO_SHOT_SCREENS).
func (u *UI) View() string    { return u.view }
func (u *UI) Overlay() string { return u.overlay }

func (u *UI) refresh() {
	if u.core == nil { // mode demo: data statis (uji render tanpa engine/jaringan)
		u.chats = demoChats()
		if u.selected == "" {
			u.selected, u.selName, u.selGroup = "2", "Keluarga", true
		}
		u.messages = demoMessages()
		u.subtitle = "online"
		u.profName = "Saya"
	} else {
		u.state = u.core.GetState()
		if !u.profFetched { // profil sendiri (avatar rail) — ambil sekali
			p := u.core.GetProfile()
			u.profName, u.selfJID, u.profFetched = p.Name, p.Jid, true
		}
		u.qr = u.core.QRCode()
		u.chats = u.core.GetChats()
		if u.selected != "" {
			u.messages = u.core.GetMessages(u.selected)
			sub := u.core.ChatSubtitle(u.selected)
			// grup: presence per-pengguna tak berlaku → tampil daftar anggota, KECUALI
			// ada yg mengetik/merekam (override). DM: tetap presence.
			if u.selGroup && !strings.Contains(sub, "mengetik") && !strings.Contains(sub, "merekam") {
				if u.groupSub == "" { // belum dihitung (mis. seleksi dipulihkan tanpa klik)
					u.groupSub = u.groupMembersSummary(u.selected)
				}
				if u.groupSub != "" {
					sub = u.groupSub
				}
			}
			u.subtitle = sub
		}
	}
	if len(u.clicks) < len(u.chats) {
		u.clicks = make([]widget.Clickable, len(u.chats))
	}
	if len(u.msgClicks) < len(u.messages) {
		u.msgClicks = make([]widget.Clickable, len(u.messages))
	}
	if len(u.quoteClicks) < len(u.messages) {
		u.quoteClicks = make([]widget.Clickable, len(u.messages))
	}
}

func demoChats() []app.ChatDTO {
	return []app.ChatDTO{
		{ID: "1", Name: "Andi Pratama", Preview: "Mantap! Sampai nanti malam 🙌", Time: "19.08", Sent: true, Status: "read", Pinned: true, Presence: "online"},
		{ID: "2", Name: "Keluarga", Preview: "Ibu: Jangan lupa makan ya nak", Time: "18.41", Group: true, Badge: 2, Unread: true},
		{ID: "3", Name: "Sarah", Preview: "Oke besok aku kabarin lagi", Time: "17.55", Badge: 1234, Unread: true}, // presence "" → tanpa titik (hidden/unknown)
		{ID: "4", Name: "Tim Proyek X", Preview: "Wohati Damar: 🏷️ Stiker", Time: "16.20", Group: true, Badge: 12, Unread: true},
		{ID: "5", Name: "Rian", Preview: "Haha iya bener banget 😄", Time: "14.03", Badge: 5, Unread: true, Muted: true, Presence: "offline"},
	}
}
func demoMessages() []app.MessageDTO {
	now := time.Now().Unix()
	yest := now - 86400
	return []app.MessageDTO{
		{ID: "m1", Dir: "in", Type: "text", Text: "Halo! Jadi nanti malam ngumpul jam berapa?", Time: "19.02", Sender: "Budi Santoso", Ts: yest},
		{ID: "m2", Dir: "out", Type: "text", Text: "Jam 8 ya, di tempat biasa 👍", Time: "19.03", Status: "delivered", Ts: yest},
		{ID: "m3", Dir: "in", Type: "text", Text: "Sip. Tempatnya yang kemarin kan?", Time: "19.05", Sender: "Budi Santoso", Ts: yest},
		{ID: "m4", Dir: "out", Type: "text", Text: "Iya betul, yang deket stasiun", Time: "19.06", Status: "read", Ts: yest, QuoteID: "m3", QuoteName: "Budi Santoso", QuoteText: "Sip. Tempatnya yang kemarin kan?", Reactions: []app.ReactionDTO{{Emoji: "👍", Count: 2}, {Emoji: "🔥", Count: 1, Mine: true}}},
		{ID: "m5", Dir: "in", Type: "text", Text: "Aku mungkin telat dikit, macet", Time: "19.40", Sender: "Citra Dewi", Ts: yest},
		{ID: "m6", Dir: "out", Type: "text", Text: "Santai, kita tunggu", Time: "19.41", Status: "read", Ts: yest, Edited: true},
		{ID: "m6b", Dir: "in", Type: "text", Time: "19.42", Sender: "Citra Dewi", Ts: yest, Revoked: true},
		{ID: "m7", Dir: "in", Type: "text", Text: "Oke sip. Aku bawa kamera sekalian foto-foto.", Time: "08.04", Sender: "Citra Dewi", Ts: now},
		{ID: "m8", Dir: "out", Type: "text", Text: "Mantap! Jangan lupa baterai cadangan", Time: "08.05", Status: "read", Ts: now},
		{ID: "m9", Dir: "in", Type: "text", Text: "Udah siap semua kok 📸", Time: "08.06", Sender: "Citra Dewi", Ts: now},
		{ID: "m10", Dir: "in", Type: "text", Text: "Btw jadi makan dulu apa langsung?", Time: "08.07", Sender: "Budi Santoso", Ts: now},
		{ID: "m11", Dir: "out", Type: "text", Text: "Makan dulu aja, laper nih 😅", Time: "08.08", Status: "delivered", Ts: now},
		{ID: "m12", Dir: "out", Type: "text", Text: "Mantap! Sampai nanti 🙌", Time: "08.09", Status: "sent", Ts: now},
		{ID: "m13", Dir: "in", Type: "image", Text: "Spot foto kemarin 📷", Time: "08.10", Sender: "Citra Dewi", Ts: now},
		{ID: "m14", Dir: "out", Type: "video", Time: "08.11", Status: "delivered", Ts: now},
		{ID: "m15", Dir: "in", Type: "poll", Text: "Liburan ke mana minggu depan?", Thumb: `["Pantai","Gunung","Kota"]`, Time: "08.12", Sender: "Budi Santoso", Ts: now},
		{ID: "m16", Dir: "in", Type: "document", Text: "Laporan_Tahunan_2026.pdf", DocMime: "application/pdf", DocSize: 524288, DocPages: 12, Time: "08.13", Sender: "Citra Dewi", Ts: now},
		{ID: "m17", Dir: "out", Type: "voice", Text: "0:12", Time: "08.14", Status: "read", Ts: now},
		{ID: "m18", Dir: "in", Type: "location", Text: "Jl. Sudirman No. 12, Jakarta", Time: "08.15", Sender: "Budi Santoso", Ts: now},
		{ID: "m19", Dir: "in", Type: "contact", Text: "Dewi Anggraini", Thumb: "+62 812-3456-7890", Time: "08.16", Sender: "Citra Dewi", Ts: now},
		{ID: "m20", Dir: "in", Type: "text", Text: "Setuju sama @Budi Santoso, nanti @Citra Dewi yang bawa kamera ya", Time: "08.17", Sender: "Rian", Ts: now, Mentions: []app.MentionDTO{{Name: "Budi Santoso"}, {Name: "Citra Dewi"}}},
		{ID: "m21", Dir: "out", Type: "text", Text: "🔥👍", Time: "08.18", Status: "read", Ts: now}, // emoji-saja → bubble diperbesar
		{ID: "m22", Dir: "in", Type: "text", Text: "Lucu banget 😂 https://www.tiktok.com/@user/video/123", Time: "08.19", Sender: "Budi Santoso", Ts: now},
		{ID: "m23", Dir: "out", Type: "audio", Text: "Lagu_Favorit.mp3", Thumb: "3:24", Time: "08.20", Status: "read", Ts: now},
		{ID: "m24", Dir: "in", Type: "sticker", Time: "08.21", Sender: "Rian", Ts: now},
		{ID: "m25", Dir: "out", Type: "gif", Time: "08.22", Status: "delivered", Ts: now},
		{ID: "m26", Dir: "in", Type: "text", Text: "Format: *tebal* _miring_ ~coret~ ```mono``` cek https://wa.me/123 ya", Time: "08.23", Sender: "Budi Santoso", Ts: now},
		{ID: "m27", Dir: "in", Type: "text", Sender: "Faisal TI2", Time: "08.24", Ts: now, Text: "‼️ADVOUPDATE‼️\n[INFO ADVOKASI: SURAT EDARAN PEMBAYARAN UKT SEMESTER GASAL 2026/2027]\n\nHalo sobat ILKOM ✨\nGimana nih kabarnya temen-temen? 😊\n\nBerdasarkan SE Nomor: B/7933/UN37/TM.01.00/2026 tentang Surat Edaran Pembayaran UKT Semester Gasal 2026/2027, terlampir...\n\n• Informasi Pengurangan UKT\n• Informasi Angsuran UKT\n• Timeline pembayaran UKT\n\nUntuk informasi selengkapnya, dapat dilihat pada link berikut.\nhttps://drive.google.com/file/d/1lnR0uT6apnBEIFB1xLhXQZabvmupmU6L/view?usp=drive_link\n\nDemikian informasi yang kami sampaikan, Terima kasih ✨\n\n———\n#HIMAILKOM2026\n#KabinetAstasae\n#DivisiEksternal"},
	}
}

func (u *UI) Layout(gtx layout.Context) layout.Dimensions {
	if time.Since(u.lastFetch) > 600*time.Millisecond {
		u.refresh()
		u.lastFetch = time.Now()
	}
	// latar
	paint.FillShape(gtx.Ops, u.t.Bg, clip.Rect{Max: gtx.Constraints.Max}.Op())

	// Media terpilih (dari dialog berkas, thread lain) → buka pratinjau di thread UI.
	u.pendMu.Lock()
	if u.pendHas && u.overlay == "" {
		u.overlay = "mediapreview"
	}
	u.pendMu.Unlock()

	// Titlebar custom (CSD Wayland) di atas; sisanya = body. Pada X11/headless tetap
	// digambar (tak merusak) — aksi window hanya jalan bila OnWinAction di-set.
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return u.titleBar(gtx) }),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return u.body(gtx) }),
	)
}

// body — isi di bawah titlebar: gerbang login/lock atau rail+sidebar+percakapan,
// dengan overlay popup di atasnya (terbatas area body agar titlebar tetap bisa diklik).
func (u *UI) body(gtx layout.Context) layout.Dimensions {
	// Gerbang login: engine tersambung tapi sesi belum siap → layar QR / nomor.
	if u.core != nil && u.state != "" && u.state != "ready" && u.state != "connected" {
		u.handleLogin(gtx)
		return LoginView(gtx, u.th, u.t, u.qr, u.loginCtl())
	}

	// Gerbang kunci aplikasi (PIN) → tutup UI utama sampai PIN benar.
	if u.locked {
		return u.lockScreen(gtx)
	}

	dims := layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return u.rail(gtx) }),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return u.sidebar(gtx) }),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { // garis pemisah abu sidebar↔percakapan
			wpx := gtx.Dp(1)
			h := gtx.Constraints.Max.Y
			paint.FillShape(gtx.Ops, u.t.Line, clip.Rect{Max: image.Pt(wpx, h)}.Op())
			return layout.Dimensions{Size: image.Pt(wpx, h)}
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return u.conversation(gtx) }),
	)
	if u.overlay != "" {
		u.overlayLayer(gtx)
	}
	return dims
}

// handleSettings memproses klik baris pane setelan: Tema (toggle gelap/terang)
// dan Keluar (logout engine → kembali ke layar QR).
func (u *UI) handleSettings(gtx layout.Context) {
	for u.setBack.Clicked(gtx) { // kembali dari sub-pane
		u.setSub = ""
		u.profLoaded = false
		u.aboutOpen = false
	}
	for u.setProfileClick.Clicked(gtx) { // kartu profil → sub-pane profil
		u.setSub = "profile"
		u.profLoaded = false
	}
	for u.profSave.Clicked(gtx) { // simpan nama/tentang
		if u.core != nil {
			u.core.SetMyName(strings.TrimSpace(u.profNameEd.Text()))
			u.core.SetMyAbout(strings.TrimSpace(u.profAboutEd.Text()))
		}
	}
	for u.aboutToggle.Clicked(gtx) { // chevron → buka/tutup dropdown saran
		u.aboutOpen = !u.aboutOpen
	}
	for i := range u.aboutClicks { // ketuk saran "Tentang" → isi editor + tutup dropdown
		if i >= len(aboutPresets) {
			break
		}
		if u.aboutClicks[i].Clicked(gtx) {
			u.profAboutEd.SetText(aboutPresets[i])
			u.aboutOpen = false
		}
	}
	for u.setClicks[0].Clicked(gtx) { // Akun → sub-pane
		u.setSub = "account"
	}
	for u.setClicks[1].Clicked(gtx) { // Privasi → sub-pane
		u.setSub = "privacy"
	}
	for u.setClicks[2].Clicked(gtx) { // Notifikasi → toggle aktif/nonaktif (persist)
		if u.core != nil {
			u.core.SetNotificationsOn(!u.core.NotificationsOn())
		}
	}
	for u.setClicks[4].Clicked(gtx) { // Bahasa → sub-pane pemilih
		u.setSub = "language"
	}
	for i := range langOrder { // pilih bahasa di sub-pane
		if i >= len(u.langClicks) {
			break
		}
		if u.langClicks[i].Clicked(gtx) && u.core != nil {
			u.core.SetLanguage(langOrder[i].code)
		}
	}
	for u.setClicks[3].Clicked(gtx) { // Tema → toggle gelap/terang
		u.dark = !u.dark
		u.t = newTheme(u.dark)
		if u.core != nil {
			u.core.SetThemeDark(u.dark) // persist lintas-restart
		}
	}
	for u.setClicks[5].Clicked(gtx) { // Penyimpanan → sub-pane
		u.setSub = "storage"
	}
	for u.setClicks[6].Clicked(gtx) { // Retensi → siklus 30/90/180/365/selamanya
		if u.core != nil {
			u.core.SetRetention(nextRetention(u.core.GetRetention()))
		}
	}
	for u.setClicks[7].Clicked(gtx) { // Simpan pesan dihapus (anti-delete)
		if u.core != nil {
			u.core.SetKeepDeleted(!u.core.GetKeepDeleted())
		}
	}
	for u.setClicks[8].Clicked(gtx) { // Kunci aplikasi → dialog atur/hapus PIN
		u.overlay = "pinset"
	}
	for u.setClicks[9].Clicked(gtx) { // Bantuan → sub-pane
		u.setSub = "help"
	}
	for u.setClicks[10].Clicked(gtx) { // Keluar
		if u.core != nil {
			u.core.Logout()
			u.state = "qr"
		}
	}
}

// loginCtl membangun controller login dari state UI (utk LoginView interaktif).
func (u *UI) loginCtl() *LoginCtl {
	return &LoginCtl{
		PhoneMode: u.loginPhone, Phone: &u.phoneEd,
		Switch: &u.loginSwitch, Submit: &u.loginSubmit, Code: u.pairCode,
	}
}

// handleLogin memproses event layar login: toggle QR↔nomor + minta kode pairing.
func (u *UI) handleLogin(gtx layout.Context) {
	for u.loginSwitch.Clicked(gtx) {
		u.loginPhone = !u.loginPhone
		u.pairCode = ""
	}
	req := false
	for u.loginSubmit.Clicked(gtx) {
		req = true
	}
	for {
		ev, ok := u.phoneEd.Update(gtx)
		if !ok {
			break
		}
		if _, ok := ev.(widget.SubmitEvent); ok {
			req = true
		}
	}
	if req && u.core != nil {
		if phone := strings.TrimSpace(u.phoneEd.Text()); phone != "" {
			u.pairCode = u.core.LinkWithPhone(phone)
		}
	}
}

// lockScreen — gate PIN layar penuh: ikon gembok + judul + input PIN (mask) →
// CheckAppPIN → buka. Salah → tanda merah.
func (u *UI) lockScreen(gtx layout.Context) layout.Dimensions {
	// Latar pakai Bg (bukan HeadBg) supaya kotak input PIN (SearchBg≈HeadBg) kontras.
	paint.FillShape(gtx.Ops, u.t.Bg, clip.Rect{Max: gtx.Constraints.Max}.Op())
	if t := u.pinEd.Text(); len(t) > 6 { // PIN maks 6 digit
		u.pinEd.SetText(t[:6])
	}
	for {
		ev, ok := u.pinEd.Update(gtx)
		if !ok {
			break
		}
		if _, ok := ev.(widget.SubmitEvent); ok {
			if u.core == nil || u.core.CheckAppPIN(strings.TrimSpace(u.pinEd.Text())) {
				u.locked, u.pinErr = false, false
				u.pinEd.SetText("")
			} else {
				u.pinErr = true
				u.pinEd.SetText("")
			}
		}
	}
	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Max.X = gtx.Dp(300)
		return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { return icon(gtx, "lock", 44, u.t.Accent) }),
			layout.Rigid(layout.Spacer{Height: unit.Dp(16)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				l := material.Label(u.th, 18, "Aplikasi terkunci")
				l.Color, l.Font.Weight = u.t.Text, font.Medium
				return l.Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				msg, col := "Masukkan PIN untuk membuka", u.t.Text2
				if u.pinErr {
					msg, col = "PIN salah, coba lagi", color.NRGBA{R: 0xe3, G: 0x5d, B: 0x6a, A: 0xff}
				}
				l := material.Label(u.th, 13.5, msg)
				l.Color = col
				return l.Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(18)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min.X = gtx.Dp(200)
				gtx.Constraints.Max.X = gtx.Dp(200)
				return u.gcField(gtx, &u.pinEd, "PIN")
			}),
		)
	})
}

// pinSetLayer — dialog atur/hapus PIN (dari setelan "Kunci aplikasi").
func (u *UI) pinSetLayer(gtx layout.Context) {
	has := u.core != nil && u.core.HasAppPIN()
	for u.pinCancel.Clicked(gtx) {
		u.overlay = ""
		u.pinSetEd.SetText("")
	}
	for u.pinClearBtn.Clicked(gtx) { // hapus PIN
		if u.core != nil {
			u.core.ClearAppPIN()
		}
		u.overlay = ""
	}
	if t := u.pinSetEd.Text(); len(t) > 6 { // PIN maks 6 digit
		u.pinSetEd.SetText(t[:6])
	}
	setPIN := func() { // 4-6 digit → simpan
		if p := strings.TrimSpace(u.pinSetEd.Text()); len(p) >= 4 && len(p) <= 6 && u.core != nil {
			u.core.SetAppPIN(p)
			u.pinSetEd.SetText("")
			u.overlay = ""
		}
	}
	for u.pinSetBtn.Clicked(gtx) { // tombol Atur
		setPIN()
	}
	for { // Enter di field → konfirmasi (bukan menyisip karakter)
		ev, ok := u.pinSetEd.Update(gtx)
		if !ok {
			break
		}
		if _, ok := ev.(widget.SubmitEvent); ok {
			setPIN()
		}
	}
	paint.FillShape(gtx.Ops, color.NRGBA{A: 110}, clip.Rect{Max: gtx.Constraints.Max}.Op())
	white := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X, gtx.Constraints.Max.X = gtx.Dp(340), gtx.Dp(340)
		macro := op.Record(gtx.Ops)
		dims := layout.UniformInset(unit.Dp(16)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			title := "Atur PIN kunci"
			if has {
				title = "Kunci aplikasi"
			}
			rows := []layout.FlexChild{
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					l := material.Label(u.th, 17, title)
					l.Color, l.Font.Weight = u.t.Text, font.SemiBold
					return l.Layout(gtx)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
			}
			if !has { // belum ada PIN → input untuk set
				rows = append(rows,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return u.gcField(gtx, &u.pinSetEd, "PIN baru (4-6 digit)")
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
				)
			} else {
				rows = append(rows,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						l := material.Label(u.th, 14, "PIN aktif. Hapus untuk menonaktifkan kunci.")
						l.Color = u.t.Text2
						return l.Layout(gtx)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
				)
			}
			rows = append(rows, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{Size: gtx.Constraints.Min} }),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						b := material.Button(u.th, &u.pinCancel, "Batal")
						b.Background, b.Color, b.CornerRadius, b.TextSize = u.t.Bg2, u.t.Text, unit.Dp(8), unit.Sp(14)
						return b.Layout(gtx)
					}),
					layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if has {
							b := material.Button(u.th, &u.pinClearBtn, "Hapus PIN")
							b.Background, b.Color, b.CornerRadius, b.TextSize = color.NRGBA{R: 0xe3, G: 0x5d, B: 0x6a, A: 0xff}, white, unit.Dp(8), unit.Sp(14)
							return b.Layout(gtx)
						}
						b := material.Button(u.th, &u.pinSetBtn, "Atur")
						b.Background, b.Color, b.CornerRadius, b.TextSize = u.t.Accent, white, unit.Dp(8), unit.Sp(14)
						return b.Layout(gtx)
					}),
				)
			}))
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, rows...)
		})
		call := macro.Stop()
		r := gtx.Dp(12)
		paint.FillShape(gtx.Ops, u.t.SidebarBg, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
		call.Add(gtx.Ops)
		return dims
	})
}

// overlayLayer — popup di atas app (backdrop klik → tutup). Komponen wave dipakai
// langsung; modal punya backdrop sendiri, info-drawer di-posisikan kanan.
func (u *UI) overlayLayer(gtx layout.Context) {
	for u.backdrop.Clicked(gtx) {
		u.overlay, u.infoCJID = "", "" // tutup → bersihkan sasaran intip-info kontak
	}
	// backdrop clickable penuh (di belakang isi)
	u.backdrop.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Dimensions{Size: gtx.Constraints.Max}
	})
	switch u.overlay {
	case "info":
		u.handleInfo(gtx) // aksi blokir/keluar grup
		// drawer kanan 400px + dim di kiri-nya
		paint.FillShape(gtx.Ops, color.NRGBA{A: 90}, clip.Rect{Max: gtx.Constraints.Max}.Op())
		w := gtx.Dp(400)
		off := op.Offset(image.Pt(gtx.Constraints.Max.X-w, 0)).Push(gtx.Ops)
		gtx.Constraints.Min.X, gtx.Constraints.Max.X = w, w
		InfoDrawerView(gtx, u.th, u.t, u.infoData())
		off.Pop()
	case "forward":
		u.handleForward(gtx)
		ModalsView(gtx, u.th, u.t, u.fwdCtl())
	case "msginfo":
		MsgInfoView(gtx, u.th, u.t)
	case "reaction":
		u.handleReaction(gtx)
		paint.FillShape(gtx.Ops, color.NRGBA{A: 110}, clip.Rect{Max: gtx.Constraints.Max}.Op())
		layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Max.X, gtx.Constraints.Max.Y = gtx.Dp(352), gtx.Dp(400)
			return ReactionPickerView(gtx, u.th, u.t, &RpCtl{Clicks: u.rpClicks, List: &u.rpList})
		})
	case "lightbox":
		for u.lightboxClose.Clicked(gtx) {
			u.overlay, u.lightboxMsg, u.lightboxCap = "", "", ""
		}
		for u.lightboxSave.Clicked(gtx) { // unduh → simpan ke disk (dialog native)
			if u.OnSaveMedia != nil && u.lightboxMsg != "" {
				u.OnSaveMedia(u.selected, u.lightboxMsg, "whatslite-"+u.lightboxMsg+".jpg")
			}
		}
		// backdrop redup rgba(0,0,0,.92) penuh di sini (case-level) agar menutup
		// rail+sidebar; LightboxView lalu menggambar foto/tombol di atasnya. Isian
		// ganda mengompensasi kurva alfa renderer headless (≈ tak tembus).
		lbRect := clip.Rect{Max: gtx.Constraints.Max}.Op()
		paint.FillShape(gtx.Ops, color.NRGBA{A: 235}, lbRect)
		paint.FillShape(gtx.Ops, color.NRGBA{A: 235}, lbRect)
		ctl := &LbCtl{Caption: u.lightboxCap, Close: &u.lightboxClose, Save: &u.lightboxSave}
		if u.lightboxMsg != "" {
			u.ensureMedia(u.selected, u.lightboxMsg, "image")
			u.mediaMu.Lock()
			if imgOp, ok := u.media[u.lightboxMsg]; ok {
				ctl.Img, ctl.Has = imgOp, true
			}
			u.mediaMu.Unlock()
		}
		LightboxView(gtx, u.th, u.t, ctl)
	case "picker":
		for u.pickerScrim.Clicked(gtx) { // ketuk di luar kartu → tutup
			u.overlay = ""
		}
		u.pickerScrim.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Dimensions{Size: gtx.Constraints.Max} // area scrim (di bawah kartu)
		})
		PickerView(gtx, u.th, u.t, u.stickerCtl(gtx))
	case "attach":
		u.handleAttach(gtx)
		if len(u.attachClicks) < AttachCount() {
			u.attachClicks = make([]widget.Clickable, AttachCount())
		}
		AttachMenuView(gtx, u.th, u.t, &AttachCtl{Clicks: u.attachClicks})
	case "msgctx":
		paint.FillShape(gtx.Ops, color.NRGBA{A: 90}, clip.Rect{Max: gtx.Constraints.Max}.Op())
		layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X, gtx.Constraints.Max.X = gtx.Dp(220), gtx.Dp(220)
			return u.ctxMenuView(gtx)
		})
	case "chatctx":
		paint.FillShape(gtx.Ops, color.NRGBA{A: 90}, clip.Rect{Max: gtx.Constraints.Max}.Op())
		layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X, gtx.Constraints.Max.X = gtx.Dp(240), gtx.Dp(240)
			return u.chatCtxView(gtx)
		})
	case "statusview":
		u.statusViewLayer(gtx)
	case "statusemoji":
		u.statusEmojiLayer(gtx)
	case "statuscompose":
		u.statusComposeLayer(gtx)
	case "pollcompose":
		u.pollComposeLayer(gtx)
	case "contactsend":
		u.contactSendLayer(gtx)
	case "loccompose":
		u.locComposeLayer(gtx)
	case "schedule":
		u.scheduleLayer(gtx)
	case "invitelink":
		u.inviteLinkLayer(gtx)
	case "mediapreview":
		u.mediaPreviewLayer(gtx)
	case "groupedit":
		u.groupEditLayer(gtx)
	case "groupcreate":
		u.groupCreateLayer(gtx)
	case "pinset":
		u.pinSetLayer(gtx)
	case "encryption":
		u.encryptionLayer(gtx)
	case "media":
		u.mediaGalleryLayer(gtx)
	case "disappearing":
		u.disappearingLayer(gtx)
	case "renamecontact":
		u.renameContactLayer(gtx)
	case "newcontact":
		u.newContactLayer(gtx)
	case "contactctx":
		u.contactCtxLayer(gtx)
	case "delconfirm":
		u.deleteConfirmLayer(gtx)
	case "memberctx":
		u.memberCtxLayer(gtx)
	case "joinlink":
		u.joinLinkLayer(gtx)
	}
}

// dispOptions — pilihan timer pesan sementara WhatsApp (label + detik).
var dispOptions = []struct {
	label string
	secs  int
}{
	{"Mati", 0}, {"24 jam", 86400}, {"7 hari", 604800}, {"90 hari", 7776000},
}

// dispLabel — label dari detik timer (kosong utk 0 → "Mati" tak ditampilkan di baris).
func dispLabel(secs int) string {
	for _, o := range dispOptions {
		if o.secs == secs {
			if secs == 0 {
				return "Mati"
			}
			return o.label
		}
	}
	return ""
}

// disappearingLayer — picker timer pesan sementara per-chat (Mati/24 jam/7 hari/90
// hari). Pilih → core.SetChatDisappearing + simpan label lokal + tutup.
func (u *UI) disappearingLayer(gtx layout.Context) layout.Dimensions {
	paint.FillShape(gtx.Ops, color.NRGBA{A: 130}, clip.Rect{Max: gtx.Constraints.Max}.Op())
	for u.dispClose.Clicked(gtx) {
		u.overlay = ""
	}
	cur := u.dispTimer[u.selected]
	for i := range u.dispClicks {
		if u.dispClicks[i].Clicked(gtx) {
			if u.core != nil {
				u.core.SetChatDisappearing(u.selected, dispOptions[i].secs)
			}
			u.dispTimer[u.selected] = dispOptions[i].secs
			u.overlay = ""
		}
	}
	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		w := gtx.Dp(320)
		gtx.Constraints.Min.X, gtx.Constraints.Max.X = w, w
		macro := op.Record(gtx.Ops)
		dims := layout.UniformInset(unit.Dp(18)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			children := []layout.FlexChild{
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					l := material.Label(u.th, 17, "Pesan sementara")
					l.Color, l.Font.Weight = u.t.Text, font.Medium
					return l.Layout(gtx)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					l := material.Label(u.th, 13, "Pesan baru di chat ini hilang setelah durasi terpilih.")
					l.Color = u.t.Text2
					return l.Layout(gtx)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
			}
			for i := range dispOptions {
				o, idx := dispOptions[i], i
				children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return u.dispClicks[idx].Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{Top: unit.Dp(11), Bottom: unit.Dp(11)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints.Min.X = gtx.Constraints.Max.X
							return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
								layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
									l := material.Label(u.th, 15.5, o.label)
									l.Color = u.t.Text
									return l.Layout(gtx)
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									if o.secs != cur {
										return layout.Dimensions{}
									}
									return icon(gtx, "check", 18, u.t.Accent)
								}),
							)
						})
					})
				}))
			}
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
		})
		call := macro.Stop()
		rr := gtx.Dp(12)
		paint.FillShape(gtx.Ops, u.t.Bg, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: rr, NE: rr, SE: rr, SW: rr}.Op(gtx.Ops))
		call.Add(gtx.Ops)
		return dims
	})
}

// renameContactLayer — modal edit nama kontak (label lokal, ala simpan/edit kontak
// di HP). Simpan → core.SaveContactLabel; kosong → RemoveContactLabel.
// joinLinkLayer — modal gabung grup lewat tautan: pratinjau nama (async) + tombol
// Gabung (JoinGroupLink → buka chat) / Batal.
func (u *UI) joinLinkLayer(gtx layout.Context) layout.Dimensions {
	paint.FillShape(gtx.Ops, color.NRGBA{A: 130}, clip.Rect{Max: gtx.Constraints.Max}.Op())
	for u.joinCancel.Clicked(gtx) {
		u.overlay, u.joinLink, u.joinPreview = "", "", ""
	}
	for u.joinConfirm.Clicked(gtx) {
		if u.core != nil && u.joinLink != "" {
			if jid := u.core.JoinGroupLink(u.joinLink); jid != "" {
				u.selected, u.selGroup, u.view = jid, true, "chats"
				u.selName = u.joinPreview
				u.searchEd.SetText("")
				u.core.OpenChat(jid)
				u.messages = u.core.GetMessages(jid)
			}
		}
		u.overlay, u.joinLink, u.joinPreview = "", "", ""
	}
	white := color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		w := gtx.Dp(340)
		gtx.Constraints.Min.X, gtx.Constraints.Max.X = w, w
		macro := op.Record(gtx.Ops)
		dims := layout.UniformInset(unit.Dp(20)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					l := material.Label(u.th, 17, "Gabung grup")
					l.Color, l.Font.Weight = u.t.Text, font.Medium
					return l.Layout(gtx)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					txt := "Memuat pratinjau…"
					if u.joinPreview != "" {
						txt = "Gabung ke \"" + u.joinPreview + "\"?"
					}
					l := material.Label(u.th, 14.5, txt)
					l.Color = u.t.Text2
					return l.Layout(gtx)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(18)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{Size: gtx.Constraints.Min} }),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							b := material.Button(u.th, &u.joinCancel, "Batal")
							b.Background, b.Color, b.CornerRadius, b.TextSize = u.t.Bg2, u.t.Text, unit.Dp(8), unit.Sp(14)
							return b.Layout(gtx)
						}),
						layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							b := material.Button(u.th, &u.joinConfirm, "Gabung")
							b.Background, b.Color, b.CornerRadius, b.TextSize = u.t.Accent, white, unit.Dp(8), unit.Sp(14)
							return b.Layout(gtx)
						}),
					)
				}),
			)
		})
		call := macro.Stop()
		rr := gtx.Dp(12)
		paint.FillShape(gtx.Ops, u.t.Bg, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: rr, NE: rr, SE: rr, SW: rr}.Op(gtx.Ops))
		call.Add(gtx.Ops)
		return dims
	})
}

// handleMemberCtx — aksi menu konteks anggota grup (kirim pesan / admin / keluarkan).
func (u *UI) handleMemberCtx(gtx layout.Context) {
	for u.mctMsg.Clicked(gtx) { // kirim pesan → buka DM anggota
		jid := u.mctJID
		u.overlay = ""
		if u.core != nil && strings.HasSuffix(jid, "@s.whatsapp.net") {
			u.selected, u.selGroup, u.selName = jid, false, u.mctName
			u.core.OpenChat(jid)
			u.messages = u.core.GetMessages(jid)
		}
	}
	for u.mctPromote.Clicked(gtx) { // jadikan/hapus admin
		if u.core != nil {
			act := "promote"
			if u.mctAdmin {
				act = "demote"
			}
			u.core.UpdateGroupParticipants(u.selected, []string{u.mctJID}, act)
		}
		u.overlay = ""
	}
	for u.mctRemove.Clicked(gtx) { // keluarkan dari grup
		if u.core != nil {
			u.core.UpdateGroupParticipants(u.selected, []string{u.mctJID}, "remove")
		}
		u.overlay = ""
	}
	for u.mctClose.Clicked(gtx) {
		u.overlay = ""
	}
}

// memberCtxLayer — menu konteks anggota grup: kirim pesan, (admin saja) jadikan/
// hapus admin + keluarkan. Aksi admin hanya muncul bila saya admin grup ini.
func (u *UI) memberCtxLayer(gtx layout.Context) layout.Dimensions {
	paint.FillShape(gtx.Ops, color.NRGBA{A: 130}, clip.Rect{Max: gtx.Constraints.Max}.Op())
	red := color.NRGBA{R: 0xe3, G: 0x5d, B: 0x6a, A: 0xff}
	normalUser := strings.HasSuffix(u.mctJID, "@s.whatsapp.net")
	row := func(c *widget.Clickable, label string, col color.NRGBA) layout.FlexChild {
		return layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return c.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min.X = gtx.Constraints.Max.X
				return layout.Inset{Top: unit.Dp(11), Bottom: unit.Dp(11), Left: unit.Dp(8), Right: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					l := material.Label(u.th, 15, label)
					l.Color, l.Alignment = col, text.Start
					return l.Layout(gtx)
				})
			})
		})
	}
	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		w := gtx.Dp(300)
		gtx.Constraints.Min.X, gtx.Constraints.Max.X = w, w
		macro := op.Record(gtx.Ops)
		dims := layout.UniformInset(unit.Dp(18)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			children := []layout.FlexChild{
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					l := material.Label(u.th, 16.5, u.mctName)
					l.Color, l.Font.Weight, l.MaxLines = u.t.Text, font.Medium, 1
					return l.Layout(gtx)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
			}
			if normalUser {
				children = append(children, row(&u.mctMsg, "Kirim pesan", u.t.Text))
			}
			if u.curGroupAmAdmin && normalUser {
				lbl := "Jadikan admin"
				if u.mctAdmin {
					lbl = "Hapus admin"
				}
				children = append(children,
					row(&u.mctPromote, lbl, u.t.Text),
					row(&u.mctRemove, "Keluarkan dari grup", red),
				)
			}
			children = append(children, row(&u.mctClose, "Tutup", u.t.Text2))
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
		})
		call := macro.Stop()
		rr := gtx.Dp(12)
		paint.FillShape(gtx.Ops, u.t.Bg, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: rr, NE: rr, SE: rr, SW: rr}.Op(gtx.Ops))
		call.Add(gtx.Ops)
		return dims
	})
}

// deleteConfirmLayer — dialog konfirmasi hapus pesan: "Hapus untuk saya" selalu;
// "Hapus untuk semua orang" hanya bila pesan SENDIRI (revoke). Paritas WhatsApp.
func (u *UI) deleteConfirmLayer(gtx layout.Context) layout.Dimensions {
	paint.FillShape(gtx.Ops, color.NRGBA{A: 130}, clip.Rect{Max: gtx.Constraints.Max}.Op())
	m := u.delTarget
	own := m.Dir == "out" && !m.Revoked
	del := func(everyone bool) {
		if u.core != nil && m.ID != "" {
			u.core.DeleteMessage(u.selected, m.ID, m.SenderID, true, everyone)
			u.messages = u.core.GetMessages(u.selected)
		}
		u.overlay, u.delTarget = "", app.MessageDTO{}
	}
	for u.delCancel.Clicked(gtx) {
		u.overlay, u.delTarget = "", app.MessageDTO{}
	}
	for u.delForMe.Clicked(gtx) {
		del(false)
	}
	for u.delForAll.Clicked(gtx) {
		del(true)
	}
	red := color.NRGBA{R: 0xe3, G: 0x5d, B: 0x6a, A: 0xff}
	btn := func(c *widget.Clickable, label string, col color.NRGBA) layout.FlexChild {
		return layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return c.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min.X = gtx.Constraints.Max.X
				return layout.Inset{Top: unit.Dp(11), Bottom: unit.Dp(11), Left: unit.Dp(8), Right: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					l := material.Label(u.th, 15, label)
					l.Color, l.Alignment = col, text.Start
					return l.Layout(gtx)
				})
			})
		})
	}
	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		w := gtx.Dp(320)
		gtx.Constraints.Min.X, gtx.Constraints.Max.X = w, w
		macro := op.Record(gtx.Ops)
		dims := layout.UniformInset(unit.Dp(18)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			children := []layout.FlexChild{
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					l := material.Label(u.th, 16.5, "Hapus pesan?")
					l.Color, l.Font.Weight = u.t.Text, font.Medium
					return l.Layout(gtx)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(14)}.Layout),
			}
			if own {
				children = append(children, btn(&u.delForAll, "Hapus untuk semua orang", red))
			}
			children = append(children,
				btn(&u.delForMe, "Hapus untuk saya", u.t.Text),
				btn(&u.delCancel, "Batal", u.t.Text2),
			)
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
		})
		call := macro.Stop()
		rr := gtx.Dp(12)
		paint.FillShape(gtx.Ops, u.t.Bg, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: rr, NE: rr, SE: rr, SW: rr}.Op(gtx.Ops))
		call.Add(gtx.Ops)
		return dims
	})
}

func (u *UI) renameContactLayer(gtx layout.Context) layout.Dimensions {
	paint.FillShape(gtx.Ops, color.NRGBA{A: 130}, clip.Rect{Max: gtx.Constraints.Max}.Op())
	target := u.renameTarget // sasaran rename (kontak intip/ctx); "" → chat terpilih
	if target == "" {
		target = u.selected
	}
	save := func() {
		nm := strings.TrimSpace(u.renameEd.Text())
		if u.core != nil {
			if nm == "" {
				u.core.RemoveContactLabel(target)
			} else {
				u.core.SaveContactLabel(target, nm)
				if target == u.selected {
					u.selName = nm
				}
			}
		}
		u.cgCache = nil // daftar kontak refresh nama
		u.overlay, u.renameTarget = "", ""
	}
	for u.renameCancel.Clicked(gtx) {
		u.overlay, u.renameTarget = "", ""
	}
	for u.renameSave.Clicked(gtx) {
		save()
	}
	for {
		ev, ok := u.renameEd.Update(gtx)
		if !ok {
			break
		}
		if _, ok := ev.(widget.SubmitEvent); ok {
			save()
		}
	}
	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		w := gtx.Dp(340)
		gtx.Constraints.Min.X, gtx.Constraints.Max.X = w, w
		macro := op.Record(gtx.Ops)
		dims := layout.UniformInset(unit.Dp(18)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					l := material.Label(u.th, 17, "Edit nama kontak")
					l.Color, l.Font.Weight = u.t.Text, font.Medium
					return l.Layout(gtx)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					mac := op.Record(gtx.Ops)
					fd := layout.Inset{Top: unit.Dp(9), Bottom: unit.Dp(9), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						gtx.Constraints.Min.X = gtx.Constraints.Max.X
						e := material.Editor(u.th, &u.renameEd, "Nama kontak")
						e.Color, e.HintColor, e.TextSize = u.t.Text, u.t.Text2, unit.Sp(15)
						return e.Layout(gtx)
					})
					call := mac.Stop()
					r := gtx.Dp(8)
					paint.FillShape(gtx.Ops, u.t.SearchBg, clip.RRect{Rect: image.Rectangle{Max: fd.Size}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
					call.Add(gtx.Ops)
					return fd
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(14)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{Size: gtx.Constraints.Min} }),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return u.renameCancel.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									l := material.Label(u.th, 14.5, "Batal")
									l.Color = u.t.Text2
									return l.Layout(gtx)
								})
							})
						}),
						layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return u.renameSave.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									l := material.Label(u.th, 14.5, "Simpan")
									l.Color, l.Font.Weight = u.t.Accent, font.Medium
									return l.Layout(gtx)
								})
							})
						}),
					)
				}),
			)
		})
		call := macro.Stop()
		rr := gtx.Dp(12)
		paint.FillShape(gtx.Ops, u.t.Bg, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: rr, NE: rr, SE: rr, SW: rr}.Op(gtx.Ops))
		call.Add(gtx.Ops)
		return dims
	})
}

// newContactLayer — modal "Kontak baru": nama + nomor → AddContact (verifikasi
// IsOnWhatsApp). Galat nomor tak terdaftar → tampil di ncErr (merah).
func (u *UI) newContactLayer(gtx layout.Context) layout.Dimensions {
	paint.FillShape(gtx.Ops, color.NRGBA{A: 130}, clip.Rect{Max: gtx.Constraints.Max}.Op())
	save := func() {
		if u.core == nil {
			u.overlay = ""
			return
		}
		jid := u.core.AddContact(u.ncName.Text(), u.ncPhone.Text())
		if jid == "" {
			u.ncErr = "Nomor tidak terdaftar di WhatsApp"
			return
		}
		u.cgCache = nil // paksa rebuild daftar kontak
		u.overlay, u.ncErr = "", ""
	}
	for u.ncCancel.Clicked(gtx) {
		u.overlay, u.ncErr = "", ""
	}
	for u.ncSave.Clicked(gtx) {
		save()
	}
	for {
		ev, ok := u.ncPhone.Update(gtx)
		if !ok {
			break
		}
		if _, ok := ev.(widget.SubmitEvent); ok {
			save()
		}
	}
	field := func(gtx layout.Context, ed *widget.Editor, hint string) layout.Dimensions {
		mac := op.Record(gtx.Ops)
		fd := layout.Inset{Top: unit.Dp(9), Bottom: unit.Dp(9), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			e := material.Editor(u.th, ed, hint)
			e.Color, e.HintColor, e.TextSize = u.t.Text, u.t.Text2, unit.Sp(15)
			return e.Layout(gtx)
		})
		call := mac.Stop()
		r := gtx.Dp(8)
		paint.FillShape(gtx.Ops, u.t.SearchBg, clip.RRect{Rect: image.Rectangle{Max: fd.Size}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
		call.Add(gtx.Ops)
		return fd
	}
	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		w := gtx.Dp(340)
		gtx.Constraints.Min.X, gtx.Constraints.Max.X = w, w
		macro := op.Record(gtx.Ops)
		dims := layout.UniformInset(unit.Dp(18)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					l := material.Label(u.th, 17, "Kontak baru")
					l.Color, l.Font.Weight = u.t.Text, font.Medium
					return l.Layout(gtx)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions { return field(gtx, &u.ncName, "Nama") }),
				layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions { return field(gtx, &u.ncPhone, "Nomor (mis. 628123456789)") }),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if u.ncErr == "" {
						return layout.Dimensions{}
					}
					return layout.Inset{Top: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						l := material.Label(u.th, 13, u.ncErr)
						l.Color = color.NRGBA{R: 0xe3, G: 0x5d, B: 0x6a, A: 0xff}
						return l.Layout(gtx)
					})
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(14)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{Size: gtx.Constraints.Min} }),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return u.ncCancel.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									l := material.Label(u.th, 14.5, "Batal")
									l.Color = u.t.Text2
									return l.Layout(gtx)
								})
							})
						}),
						layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return u.ncSave.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									l := material.Label(u.th, 14.5, "Simpan")
									l.Color, l.Font.Weight = u.t.Accent, font.Medium
									return l.Layout(gtx)
								})
							})
						}),
					)
				}),
			)
		})
		call := macro.Stop()
		rr := gtx.Dp(12)
		paint.FillShape(gtx.Ops, u.t.Bg, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: rr, NE: rr, SE: rr, SW: rr}.Op(gtx.Ops))
		call.Add(gtx.Ops)
		return dims
	})
}

// contactCtxLayer — menu konteks kontak (klik-kanan di pane Kontak): Kirim pesan,
// Info kontak, Edit nama, Blokir/Buka blokir, Hapus kontak. Aksi pada u.cctContact.
func (u *UI) contactCtxLayer(gtx layout.Context) layout.Dimensions {
	c := u.cctContact
	if c.JID == "" {
		u.overlay = ""
		return layout.Dimensions{}
	}
	paint.FillShape(gtx.Ops, color.NRGBA{A: 90}, clip.Rect{Max: gtx.Constraints.Max}.Op())
	blocked := u.core != nil && u.core.IsBlocked(c.JID)
	blockLbl, blockIcon := "Blokir kontak", "block"
	if blocked {
		blockLbl = "Buka blokir"
	}
	type ctItem struct {
		c      *widget.Clickable
		icon   string
		label  string
		danger bool
		do     func()
	}
	openChat := func() {
		u.selected, u.selName, u.selGroup = c.JID, c.Name, false
		u.view = "chats"
		if u.core != nil {
			u.core.OpenChat(c.JID)
			u.messages = u.core.GetMessages(c.JID)
			u.prefetchHistory(c.JID)
		}
		u.msgList.ScrollTo(len(u.messages))
	}
	items := []ctItem{
		{&u.cctMsg, "message", "Kirim pesan", false, openChat},
		{&u.cctInfo, "info", "Info kontak", false, func() {
			u.infoCJID, u.infoCName = c.JID, c.Name // intip info tanpa buka chat
			u.overlay = "info"
		}},
		{&u.cctRename, "editpen", "Edit nama", false, func() {
			u.renameEd.SetText(c.Name)
			u.renameTarget = c.JID // sasaran rename tanpa membuka chat
			u.overlay = "renamecontact"
		}},
		{&u.cctBlock, blockIcon, blockLbl, !blocked, func() {
			if u.core != nil {
				u.core.Block(c.JID, !blocked)
			}
			u.overlay = ""
		}},
		{&u.cctDelete, "trash", "Hapus kontak", true, func() {
			if u.core != nil {
				u.core.DeleteContact(c.JID)
			}
			u.cgCache = nil
			u.overlay = ""
		}},
	}
	for i := range items {
		it := items[i]
		for it.c.Clicked(gtx) {
			it.do()
			if u.overlay == "contactctx" { // aksi yg tak set overlay sendiri → tutup
				u.overlay = ""
			}
		}
	}
	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		w := gtx.Dp(240)
		gtx.Constraints.Min.X, gtx.Constraints.Max.X = w, w
		children := make([]layout.FlexChild, 0, len(items)+1)
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(11), Bottom: unit.Dp(7), Left: unit.Dp(14), Right: unit.Dp(14)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				l := material.Label(u.th, 13.5, c.Name)
				l.Color, l.Font.Weight, l.MaxLines = u.t.Text2, font.Medium, 1
				return l.Layout(gtx)
			})
		}))
		for i := range items {
			it := items[i]
			children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return it.c.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Top: unit.Dp(9), Bottom: unit.Dp(9), Left: unit.Dp(14), Right: unit.Dp(14)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						gtx.Constraints.Min.X = gtx.Constraints.Max.X
						dcol := u.t.Text
						if it.danger {
							dcol = color.NRGBA{R: 0xe3, G: 0x5d, B: 0x6a, A: 0xff}
						}
						return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions { return icon(gtx, it.icon, 18, dcol) }),
							layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								lbl := material.Label(u.th, 14.5, it.label)
								lbl.Color = dcol
								return lbl.Layout(gtx)
							}),
						)
					})
				})
			}))
		}
		macro := op.Record(gtx.Ops)
		dims := layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
		call := macro.Stop()
		rr := gtx.Dp(10)
		paint.FillShape(gtx.Ops, u.t.Bg, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: rr, NE: rr, SE: rr, SW: rr}.Op(gtx.Ops))
		call.Add(gtx.Ops)
		return dims
	})
}

// encryptionLayer — kartu info enkripsi end-to-end (paritas WhatsApp "Enkripsi").
func (u *UI) encryptionLayer(gtx layout.Context) layout.Dimensions {
	paint.FillShape(gtx.Ops, color.NRGBA{A: 130}, clip.Rect{Max: gtx.Constraints.Max}.Op())
	for u.encClose.Clicked(gtx) {
		u.overlay = ""
	}
	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		w := gtx.Dp(360)
		gtx.Constraints.Min.X, gtx.Constraints.Max.X = w, w
		macro := op.Record(gtx.Ops)
		dims := layout.UniformInset(unit.Dp(22)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = gtx.Constraints.Max.X
					return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions { return icon(gtx, "lock", 40, u.t.Accent) })
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					l := material.Label(u.th, 17, "Terenkripsi end-to-end")
					l.Color, l.Font.Weight, l.Alignment = u.t.Text, font.Medium, text.Middle
					gtx.Constraints.Min.X = gtx.Constraints.Max.X
					return l.Layout(gtx)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					l := material.Label(u.th, 13.5, "Pesan dan panggilan di chat ini dilindungi enkripsi end-to-end. Tidak ada pihak di luar chat ini—termasuk WhatsApp—yang dapat membacanya.")
					l.Color, l.Alignment = u.t.Text2, text.Middle
					return l.Layout(gtx)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(16)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return u.encClose.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							l := material.Label(u.th, 14.5, "Tutup")
							l.Color, l.Font.Weight = u.t.Accent, font.Medium
							return l.Layout(gtx)
						})
					})
				}),
			)
		})
		call := macro.Stop()
		rr := gtx.Dp(12)
		paint.FillShape(gtx.Ops, u.t.Bg, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: rr, NE: rr, SE: rr, SW: rr}.Op(gtx.Ops))
		call.Add(gtx.Ops)
		return dims
	})
}

// mediaGalleryLayer — galeri "Media, tautan, dokumen": grid foto/video chat aktif
// (core.GetChatMedia). Ketuk → lightbox. Backdrop/tutup di tepi.
func (u *UI) mediaGalleryLayer(gtx layout.Context) layout.Dimensions {
	paint.FillShape(gtx.Ops, color.NRGBA{A: 150}, clip.Rect{Max: gtx.Constraints.Max}.Op())
	var media []app.MessageDTO
	if u.core != nil {
		media = u.core.GetChatMedia(u.selected)
	}
	for u.encClose.Clicked(gtx) { // pakai ulang encClose sbg tombol tutup galeri
		u.overlay = ""
	}
	if len(u.mediaCellClicks) < len(media) {
		u.mediaCellClicks = make([]widget.Clickable, len(media))
	}
	for i := range media {
		if i < len(u.mediaCellClicks) && u.mediaCellClicks[i].Clicked(gtx) {
			u.lightboxMsg, u.lightboxCap, u.overlay = media[i].ID, media[i].Text, "lightbox"
		}
	}
	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		w, h := gtx.Dp(460), gtx.Dp(560)
		gtx.Constraints.Min, gtx.Constraints.Max = image.Pt(w, h), image.Pt(w, h)
		macro := op.Record(gtx.Ops)
		dims := layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.UniformInset(unit.Dp(16)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							l := material.Label(u.th, 17, "Media, tautan, dokumen")
							l.Color, l.Font.Weight = u.t.Text, font.Medium
							return l.Layout(gtx)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return u.encClose.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return icon(gtx, "close", 22, u.t.Text2)
							})
						}),
					)
				})
			}),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				if len(media) == 0 {
					gtx.Constraints.Min = gtx.Constraints.Max
					return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						l := material.Label(u.th, 14, "Belum ada media")
						l.Color = u.t.Text2
						return l.Layout(gtx)
					})
				}
				return material.List(u.th, &u.mediaGalList).Layout(gtx, (len(media)+2)/3, func(gtx layout.Context, row int) layout.Dimensions {
					return u.mediaGalleryRow(gtx, media, row)
				})
			}),
		)
		call := macro.Stop()
		rr := gtx.Dp(12)
		paint.FillShape(gtx.Ops, u.t.Bg, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: rr, NE: rr, SE: rr, SW: rr}.Op(gtx.Ops))
		call.Add(gtx.Ops)
		return dims
	})
}

// mediaGalleryRow — satu baris 3 sel grid media (thumbnail kotak, ketuk → lightbox).
func (u *UI) mediaGalleryRow(gtx layout.Context, media []app.MessageDTO, row int) layout.Dimensions {
	cell := func(gtx layout.Context, idx int) layout.Dimensions {
		side := (gtx.Constraints.Max.X - gtx.Dp(8)) / 3
		bsz := image.Pt(side, side)
		if idx >= len(media) {
			return layout.Dimensions{Size: bsz}
		}
		m := media[idx]
		u.ensureMedia(u.selected, m.ID, m.Type)
		u.mediaMu.Lock()
		op, ok := u.media[m.ID]
		u.mediaMu.Unlock()
		return u.mediaCellClicks[idx].Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min, gtx.Constraints.Max = bsz, bsz
			r := gtx.Dp(4)
			cl := clip.RRect{Rect: image.Rectangle{Max: bsz}, NW: r, NE: r, SE: r, SW: r}.Push(gtx.Ops)
			if ok {
				op.Add(gtx.Ops)
				paint.PaintOp{}.Add(gtx.Ops)
			} else {
				paint.FillShape(gtx.Ops, u.t.Bg2, clip.Rect{Max: bsz}.Op())
				layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions { return icon(gtx, "media", 28, u.t.Text2) })
			}
			cl.Pop()
			return layout.Dimensions{Size: bsz}
		})
	}
	return layout.Inset{Left: unit.Dp(8), Right: unit.Dp(8), Bottom: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { return cell(gtx, row*3) }),
			layout.Rigid(layout.Spacer{Width: unit.Dp(4)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { return cell(gtx, row*3+1) }),
			layout.Rigid(layout.Spacer{Width: unit.Dp(4)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { return cell(gtx, row*3+2) }),
		)
	})
}

// doCtxAction menjalankan aksi context-menu pesan terhadap engine. Bintangi/Hapus
// langsung; Balas mengaktifkan banner balas di composer (kirim → core.Reply).
func (u *UI) doCtxAction(gtx layout.Context, label string) {
	m := u.ctxMsg // snapshot saat menu dibuka (bukan index yg bisa bergeser)
	if m.ID == "" {
		return
	}
	fromMe := m.Dir == "out"
	switch label {
	case "Bintangi":
		if u.core != nil {
			u.core.StarMessage(u.selected, m.ID, m.SenderID, fromMe, true)
		}
	case "Hapus":
		u.delTarget = m // buka dialog pilih: hapus untuk saya / untuk semua (overlay delconfirm)
	case "Balas":
		u.replyTo, u.replyName, u.replyText = m.ID, u.replyDisplayName(m), m.Text
	case "Reaksi":
		u.reactMsgID, u.reactSender, u.reactFromMe = m.ID, m.SenderID, fromMe // target reaksi
	case "Teruskan":
		u.fwdMsgID = m.ID // pesan sumber utk diteruskan (sumber = u.selected)
	case "Unduh":
		if u.OnSaveMedia != nil {
			u.OnSaveMedia(u.selected, m.ID, saveName(m))
		}
	case "Edit":
		u.clearReply() // tak bisa edit + balas sekaligus
		u.editTarget, u.editText = m.ID, m.Text
		u.editor.SetText(m.Text)
	case "Sematkan", "Lepas sematan":
		if u.core != nil {
			u.core.PinMessage(u.selected, m.ID, m.SenderID, fromMe, label == "Sematkan")
			u.pinnedAt = time.Time{} // paksa muat ulang cache
		}
	case "Pilih":
		u.selMode = true
		u.selSet[m.ID] = true
	case "Terjemahkan":
		u.ensureTranslate(m.ID, m.Text)
	case "Sembunyikan terjemahan":
		u.transMu.Lock()
		delete(u.transText, m.ID)
		delete(u.transTried, m.ID)
		u.transMu.Unlock()
	case "Salin":
		gtx.Execute(clipboard.WriteCmd{Type: "application/text", Data: io.NopCloser(strings.NewReader(m.Text))})
	}
}

// toggleSel — pilih/lepas pesan di mode-pilih; kosong → keluar mode.
func (u *UI) toggleSel(id string) {
	if u.selSet[id] {
		delete(u.selSet, id)
	} else {
		u.selSet[id] = true
	}
	if len(u.selSet) == 0 {
		u.selMode = false
	}
}

// exitSel — keluar mode-pilih + bersihkan pilihan.
func (u *UI) exitSel() {
	u.selMode = false
	for k := range u.selSet {
		delete(u.selSet, k)
	}
}

// deleteSelected — hapus semua pesan terpilih (everyone utk pesan sendiri, else
// hanya-saya), refresh, lalu keluar mode-pilih.
func (u *UI) deleteSelected() {
	if u.core != nil {
		for i := range u.messages {
			m := u.messages[i]
			if u.selSet[m.ID] {
				own := m.Dir == "out"
				u.core.DeleteMessage(u.selected, m.ID, m.SenderID, own, own)
			}
		}
		u.messages = u.core.GetMessages(u.selected)
	}
	u.exitSel()
}

// forwardSelected — siapkan teruskan-banyak (urut pesan) lalu buka modal tujuan.
func (u *UI) forwardSelected() {
	u.fwdMsgIDs = u.fwdMsgIDs[:0]
	for i := range u.messages {
		if u.selSet[u.messages[i].ID] {
			u.fwdMsgIDs = append(u.fwdMsgIDs, u.messages[i].ID)
		}
	}
	u.exitSel()
	if len(u.fwdMsgIDs) > 0 {
		u.overlay = "forward"
	}
}

// saveName — nama berkas saran utk simpan media (ekstensi sesuai tipe).
func saveName(m app.MessageDTO) string {
	ext := ".bin"
	switch m.Type {
	case "image":
		ext = ".jpg"
	case "video":
		ext = ".mp4"
	case "gif":
		ext = ".gif"
	case "voice", "audio", "ptt":
		ext = ".ogg"
	case "sticker":
		ext = ".webp"
	case "document":
		ext = "" // pakai nama dokumen asli bila ada
		if m.Text != "" {
			return "whatslite-" + m.Text
		}
	}
	return "whatslite-" + m.ID + ext
}

// handleForward memproses modal teruskan: ketuk chat tujuan → core.Forward; batal.
func (u *UI) handleForward(gtx layout.Context) {
	if len(u.fwdClicks) < len(u.chats) {
		u.fwdClicks = make([]widget.Clickable, len(u.chats))
	}
	for u.fwdCancel.Clicked(gtx) {
		u.closeForward()
	}
	for i := range u.chats { // ketuk baris → pilih/lepas tujuan (multi-target)
		if i >= len(u.fwdClicks) {
			break
		}
		for u.fwdClicks[i].Clicked(gtx) {
			id := u.chats[i].ID
			if u.fwdSel[id] {
				delete(u.fwdSel, id)
			} else {
				u.fwdSel[id] = true
			}
		}
	}
	for u.fwdSend.Clicked(gtx) { // Kirim → teruskan ke SEMUA tujuan terpilih
		src := u.selected
		if u.fwdSrc != "" { // forward dari status → sumber status@broadcast
			src = u.fwdSrc
		}
		if u.core != nil {
			ids := u.fwdMsgIDs
			if len(ids) == 0 && u.fwdMsgID != "" {
				ids = []string{u.fwdMsgID}
			}
			for dst := range u.fwdSel {
				for _, id := range ids {
					u.core.Forward(src, id, dst)
				}
			}
		}
		u.closeForward()
	}
}

// closeForward — tutup modal teruskan + bersihkan state pilihan/sumber.
func (u *UI) closeForward() {
	u.fwdMsgID, u.fwdMsgIDs, u.fwdSrc = "", nil, ""
	for k := range u.fwdSel {
		delete(u.fwdSel, k)
	}
	u.overlay = ""
}

// handleAttach memproses klik menu lampiran → OnAttach(chat, kategori) lalu tutup.
func (u *UI) handleAttach(gtx layout.Context) {
	if len(u.attachClicks) < AttachCount() {
		u.attachClicks = make([]widget.Clickable, AttachCount())
	}
	for i := range u.attachClicks {
		for u.attachClicks[i].Clicked(gtx) {
			cat := AttachCategory(i)
			if cat == "poll" { // polling disusun in-app (input teks) → SendPoll
				u.overlay = "pollcompose"
				continue
			}
			if cat == "contact" { // pilih kontak → SendContact
				u.overlay = "contactsend"
				continue
			}
			if cat == "location" { // input tempat+koordinat → SendLocation
				u.overlay = "loccompose"
				continue
			}
			if cat == "schedule" { // jadwalkan teks composer saat ini
				if strings.TrimSpace(u.editor.Text()) != "" {
					u.overlay = "schedule"
				} else {
					u.overlay = ""
				}
				continue
			}
			if u.OnAttach != nil && u.selected != "" {
				u.OnAttach(u.selected, cat) // media/document via dialog berkas; contact/location TODO
			}
			u.overlay = ""
		}
	}
}

// groupCreateLayer — modal buat grup: nama + daftar kontak multi-pilih + Buat.
func (u *UI) groupCreateLayer(gtx layout.Context) {
	if u.core != nil && u.contactSendCache == nil { // pakai cache kontak yg sama
		u.contactSendCache = u.core.GetContacts()
		sort.Slice(u.contactSendCache, func(i, j int) bool {
			return strings.ToLower(u.contactSendCache[i].Name) < strings.ToLower(u.contactSendCache[j].Name)
		})
	}
	addMode := u.gcMode == "addmember"
	existing := map[string]bool{} // anggota yg sudah ada → disembunyikan (addmember)
	if addMode {
		for _, j := range u.infoMemberJIDs {
			existing[j] = true
		}
	}
	visible := make([]int, 0, len(u.contactSendCache)) // indeks kontak yg tampil
	for i, c := range u.contactSendCache {
		if !existing[c.JID] {
			visible = append(visible, i)
		}
	}
	if len(u.gcClicks) < len(u.contactSendCache) {
		u.gcClicks = make([]widget.Clickable, len(u.contactSendCache))
	}
	resetGC := func() {
		u.overlay, u.contactSendCache = "", nil
		u.gcNameEd.SetText("")
		u.gcSel = map[string]bool{}
		u.gcMode, u.gcGroupJID = "", ""
	}
	for u.gcCancel.Clicked(gtx) {
		resetGC()
	}
	for i := range u.contactSendCache {
		if i >= len(u.gcClicks) {
			break
		}
		for u.gcClicks[i].Clicked(gtx) { // toggle pilih
			j := u.contactSendCache[i].JID
			u.gcSel[j] = !u.gcSel[j]
		}
	}
	for u.gcCreate.Clicked(gtx) {
		name := strings.TrimSpace(u.gcNameEd.Text())
		var members []string
		for _, c := range u.contactSendCache {
			if u.gcSel[c.JID] {
				members = append(members, c.JID)
			}
		}
		if addMode {
			if len(members) >= 1 && u.core != nil {
				u.core.UpdateGroupParticipants(u.gcGroupJID, members, "add")
			}
		} else if name != "" && len(members) >= 1 && u.core != nil {
			if jid := u.core.CreateGroup(name, members); jid != "" {
				u.selected, u.selName, u.selGroup = jid, name, true
				u.view = "chats"
				u.core.OpenChat(jid)
				u.messages = u.core.GetMessages(jid)
			}
		}
		resetGC()
	}
	u.gcList.Axis = layout.Vertical
	selCount := 0
	for _, v := range u.gcSel {
		if v {
			selCount++
		}
	}
	paint.FillShape(gtx.Ops, color.NRGBA{A: 110}, clip.Rect{Max: gtx.Constraints.Max}.Op())
	layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		w, h := gtx.Dp(390), gtx.Dp(520)
		gtx.Constraints.Min.X, gtx.Constraints.Max.X = w, w
		gtx.Constraints.Max.Y = h
		macro := op.Record(gtx.Ops)
		dims := layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				title := "Grup baru"
				if addMode {
					title = "Tambah anggota"
				}
				return layout.Inset{Top: unit.Dp(14), Left: unit.Dp(16), Right: unit.Dp(16), Bottom: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					l := material.Label(u.th, 17, title)
					l.Color, l.Font.Weight = u.t.Text, font.SemiBold
					return l.Layout(gtx)
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { // nama grup (hanya buat-grup)
				if addMode {
					return layout.Dimensions{}
				}
				return layout.Inset{Left: unit.Dp(16), Right: unit.Dp(16), Bottom: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return u.gcField(gtx, &u.gcNameEd, "Nama grup")
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Left: unit.Dp(16), Bottom: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					l := material.Label(u.th, 12.5, itoa(selCount)+" anggota dipilih")
					l.Color = u.t.Text2
					return l.Layout(gtx)
				})
			}),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				if len(visible) == 0 {
					return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						msg := "Tak ada kontak"
						if addMode {
							msg = "Semua kontak sudah jadi anggota"
						}
						l := material.Label(u.th, 14, msg)
						l.Color = u.t.Text2
						return l.Layout(gtx)
					})
				}
				return material.List(u.th, &u.gcList).Layout(gtx, len(visible), func(gtx layout.Context, vi int) layout.Dimensions {
					i := visible[vi]
					c := u.contactSendCache[i]
					return u.gcClicks[i].Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return u.gcContactRow(gtx, c, u.gcSel[c.JID])
					})
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(14), Left: unit.Dp(16), Right: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = gtx.Constraints.Max.X
					return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{Size: gtx.Constraints.Min} }),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							b := material.Button(u.th, &u.gcCancel, "Batal")
							b.Background, b.Color, b.CornerRadius, b.TextSize = u.t.Bg2, u.t.Text, unit.Dp(8), unit.Sp(14)
							return b.Layout(gtx)
						}),
						layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							okLbl := "Buat"
							if addMode {
								okLbl = "Tambah"
							}
							b := material.Button(u.th, &u.gcCreate, okLbl)
							b.Background, b.Color, b.CornerRadius, b.TextSize = u.t.Accent, color.NRGBA{R: 255, G: 255, B: 255, A: 255}, unit.Dp(8), unit.Sp(14)
							return b.Layout(gtx)
						}),
					)
				})
			}),
		)
		call := macro.Stop()
		r := gtx.Dp(12)
		paint.FillShape(gtx.Ops, u.t.SidebarBg, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
		call.Add(gtx.Ops)
		return dims
	})
}

func (u *UI) gcField(gtx layout.Context, ed *widget.Editor, hint string) layout.Dimensions {
	macro := op.Record(gtx.Ops)
	dims := layout.Inset{Top: unit.Dp(9), Bottom: unit.Dp(9), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		e := material.Editor(u.th, ed, hint)
		e.Color, e.HintColor, e.TextSize = u.t.Text, u.t.Text2, unit.Sp(14.5)
		return e.Layout(gtx)
	})
	call := macro.Stop()
	r := gtx.Dp(8)
	paint.FillShape(gtx.Ops, u.t.SearchBg, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
	call.Add(gtx.Ops)
	return dims
}

// gcContactRow — baris kontak buat-grup: avatar + nama + tanda centang bila dipilih.
func (u *UI) gcContactRow(gtx layout.Context, c app.ContactRowDTO, sel bool) layout.Dimensions {
	return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(16), Right: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { return u.avatar(gtx, c.Name, c.JID, 38) }),
			layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				l := material.Label(u.th, 15, c.Name)
				l.Color, l.MaxLines = u.t.Text, 1
				return l.Layout(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				d := gtx.Dp(22)
				sz := image.Pt(d, d)
				if sel {
					paint.FillShape(gtx.Ops, u.t.Accent, clip.Ellipse{Max: sz}.Op(gtx.Ops))
					gtx.Constraints.Min, gtx.Constraints.Max = sz, sz
					layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return icon(gtx, "check", 14, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
					})
				} else {
					paint.FillShape(gtx.Ops, u.t.Line, clip.Ellipse{Max: sz}.Op(gtx.Ops))
					bw := gtx.Dp(2)
					in := image.Rectangle{Min: image.Pt(bw, bw), Max: image.Pt(d-bw, d-bw)}
					paint.FillShape(gtx.Ops, u.t.SidebarBg, clip.Ellipse{Min: in.Min, Max: in.Max}.Op(gtx.Ops))
				}
				return layout.Dimensions{Size: sz}
			}),
		)
	})
}

// stickerCtl membangun controller picker stiker (data nyata) + tangani klik kirim.
// nil bila demo (core nil) → grid placeholder.
func (u *UI) stickerCtl(gtx layout.Context) *PkCtl {
	if u.core == nil {
		return nil
	}
	if u.stickerCache == nil {
		u.stickerCache = u.core.ListSavedStickers()
	}
	if len(u.stickerClicks) < len(u.stickerCache) {
		u.stickerClicks = make([]widget.Clickable, len(u.stickerCache))
	}
	items := make([]PkItem, len(u.stickerCache))
	for i, s := range u.stickerCache {
		u.ensureStickerThumb(s.Hash)
		u.photoMu.Lock()
		op, ok := u.stickerThumbs[s.Hash]
		u.photoMu.Unlock()
		if ok {
			items[i] = PkItem{Thumb: op, Has: true}
		}
		if i < len(u.stickerClicks) {
			for u.stickerClicks[i].Clicked(gtx) { // tap stiker → kirim
				if u.selected != "" {
					u.core.SendSavedSticker(u.selected, s.Hash)
					u.messages = u.core.GetMessages(u.selected)
				}
				u.overlay, u.stickerCache = "", nil
				u.msgList.ScrollTo(len(u.messages))
			}
		}
	}
	return &PkCtl{Items: items, Clicks: u.stickerClicks}
}

// ensureStickerThumb memuat byte stiker (StickerBytes) sekali per hash → decode
// (webp) → cache ImageOp. Async agar tak memblok UI.
func (u *UI) ensureStickerThumb(hash string) {
	if hash == "" || u.stickerTried[hash] {
		return
	}
	u.stickerTried[hash] = true
	go func() {
		img := decodeImage(u.core.StickerBytes(hash))
		if img == nil {
			return
		}
		op := paint.NewImageOp(img)
		u.photoMu.Lock() // pakai lock yg sama (akses peta dari goroutine)
		u.stickerThumbs[hash] = op
		u.photoMu.Unlock()
	}()
}

// contactSendLayer — modal pilih kontak utk dikirim (SendContact).
func (u *UI) contactSendLayer(gtx layout.Context) {
	if u.core != nil && u.contactSendCache == nil {
		u.contactSendCache = u.core.GetContacts()
		sort.Slice(u.contactSendCache, func(i, j int) bool {
			return strings.ToLower(u.contactSendCache[i].Name) < strings.ToLower(u.contactSendCache[j].Name)
		})
	}
	if len(u.contactSendClicks) < len(u.contactSendCache) {
		u.contactSendClicks = make([]widget.Clickable, len(u.contactSendCache))
	}
	for u.contactSendCancel.Clicked(gtx) {
		u.overlay = ""
		u.contactSendCache = nil
	}
	for i := range u.contactSendCache {
		if i >= len(u.contactSendClicks) {
			break
		}
		for u.contactSendClicks[i].Clicked(gtx) {
			c := u.contactSendCache[i]
			if u.core != nil && u.selected != "" {
				u.core.SendContact(u.selected, c.Name, c.Phone)
				u.messages = u.core.GetMessages(u.selected)
			}
			u.overlay = ""
			u.contactSendCache = nil
		}
	}
	u.contactSendList.Axis = layout.Vertical
	paint.FillShape(gtx.Ops, color.NRGBA{A: 110}, clip.Rect{Max: gtx.Constraints.Max}.Op())
	layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		w, h := gtx.Dp(408), gtx.Dp(460)
		gtx.Constraints.Min.X, gtx.Constraints.Max.X = w, w
		gtx.Constraints.Max.Y = h
		macro := op.Record(gtx.Ops)
		dims := layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(16), Bottom: unit.Dp(10), Left: unit.Dp(16), Right: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							l := material.Label(u.th, 17, "Kirim kontak")
							l.Color, l.Font.Weight = u.t.Text, font.SemiBold
							return l.Layout(gtx)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return u.contactSendCancel.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return icon(gtx, "close", 20, u.t.Text2)
							})
						}),
					)
				})
			}),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				if len(u.contactSendCache) == 0 {
					return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						l := material.Label(u.th, 14, "Tak ada kontak")
						l.Color = u.t.Text2
						return l.Layout(gtx)
					})
				}
				return material.List(u.th, &u.contactSendList).Layout(gtx, len(u.contactSendCache), func(gtx layout.Context, i int) layout.Dimensions {
					c := u.contactSendCache[i]
					return u.contactSendClicks[i].Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(16), Right: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints.Min.X = gtx.Constraints.Max.X
							return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions { return u.avatar(gtx, c.Name, c.JID, 40) }),
								layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
								layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
									return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											l := material.Label(u.th, 15, c.Name)
											l.Color, l.MaxLines = u.t.Text, 1
											return l.Layout(gtx)
										}),
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											l := material.Label(u.th, 13, c.Phone)
											l.Color, l.MaxLines = u.t.Text2, 1
											return l.Layout(gtx)
										}),
									)
								}),
							)
						})
					})
				})
			}),
		)
		call := macro.Stop()
		r := gtx.Dp(12)
		paint.FillShape(gtx.Ops, u.t.SidebarBg, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
		call.Add(gtx.Ops)
		return dims
	})
}

// locComposeLayer — modal susun lokasi: nama tempat + lat + lng → SendLocation.
func (u *UI) locComposeLayer(gtx layout.Context) {
	for u.locCancel.Clicked(gtx) {
		u.overlay = ""
	}
	for u.locSend.Clicked(gtx) {
		name := strings.TrimSpace(u.locNameEd.Text())
		lat, _ := strconv.ParseFloat(strings.TrimSpace(u.locLatEd.Text()), 64)
		lng, _ := strconv.ParseFloat(strings.TrimSpace(u.locLngEd.Text()), 64)
		if u.core != nil && u.selected != "" && (name != "" || lat != 0 || lng != 0) {
			u.core.SendLocation(u.selected, lat, lng, name)
			u.locNameEd.SetText("")
			u.locLatEd.SetText("")
			u.locLngEd.SetText("")
			u.messages = u.core.GetMessages(u.selected)
		}
		u.overlay = ""
	}
	paint.FillShape(gtx.Ops, color.NRGBA{A: 110}, clip.Rect{Max: gtx.Constraints.Max}.Op())
	layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X, gtx.Constraints.Max.X = gtx.Dp(360), gtx.Dp(360)
		return composeCard(gtx, u.th, u.t, "Bagikan lokasi", []composeField{
			{&u.locNameEd, "Nama tempat (mis. Kantor)"},
			{&u.locLatEd, "Lintang (lat, mis. -6.2088)"},
			{&u.locLngEd, "Bujur (lng, mis. 106.8456)"},
		}, &u.locCancel, &u.locSend, "Kirim")
	})
}

// composeField + composeCard — kartu modal generik (judul + input + Batal/aksi).
type composeField struct {
	ed   *widget.Editor
	hint string
}

func composeCard(gtx layout.Context, th *material.Theme, t Theme, title string, fields []composeField, cancel, action *widget.Clickable, actionLabel string) layout.Dimensions {
	white := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	field := func(gtx layout.Context, ed *widget.Editor, hint string) layout.Dimensions {
		macro := op.Record(gtx.Ops)
		dims := layout.Inset{Top: unit.Dp(9), Bottom: unit.Dp(9), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			e := material.Editor(th, ed, hint)
			e.Color, e.HintColor, e.TextSize = t.Text, t.Text2, unit.Sp(14.5)
			return e.Layout(gtx)
		})
		call := macro.Stop()
		r := gtx.Dp(8)
		paint.FillShape(gtx.Ops, t.SearchBg, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
		call.Add(gtx.Ops)
		return dims
	}
	macro := op.Record(gtx.Ops)
	dims := layout.UniformInset(unit.Dp(16)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		children := []layout.FlexChild{
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				l := material.Label(th, 17, title)
				l.Color, l.Font.Weight = t.Text, font.SemiBold
				return l.Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
		}
		for i := range fields {
			f := fields[i]
			children = append(children,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions { return field(gtx, f.ed, f.hint) }),
				layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
			)
		}
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{Size: gtx.Constraints.Min} }),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					b := material.Button(th, cancel, "Batal")
					b.Background, b.Color, b.CornerRadius, b.TextSize = t.Bg2, t.Text, unit.Dp(8), unit.Sp(14)
					return b.Layout(gtx)
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					b := material.Button(th, action, actionLabel)
					b.Background, b.Color, b.CornerRadius, b.TextSize = t.Accent, white, unit.Dp(8), unit.Sp(14)
					return b.Layout(gtx)
				}),
			)
		}))
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
	})
	call := macro.Stop()
	r := gtx.Dp(12)
	paint.FillShape(gtx.Ops, t.SidebarBg, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
	call.Add(gtx.Ops)
	return dims
}

// pollComposeLayer — modal susun polling: pertanyaan + 4 opsi + Buat/Batal.
func (u *UI) pollComposeLayer(gtx layout.Context) {
	for u.pollCancel.Clicked(gtx) {
		u.overlay = ""
	}
	for u.pollCreate.Clicked(gtx) {
		q := strings.TrimSpace(u.pollQEd.Text())
		var opts []string
		for i := range u.pollOptEds {
			if o := strings.TrimSpace(u.pollOptEds[i].Text()); o != "" {
				opts = append(opts, o)
			}
		}
		if q != "" && len(opts) >= 2 && u.core != nil && u.selected != "" {
			u.core.SendPoll(u.selected, q, opts, 1)
			u.pollQEd.SetText("")
			for i := range u.pollOptEds {
				u.pollOptEds[i].SetText("")
			}
			u.messages = u.core.GetMessages(u.selected)
		}
		u.overlay = ""
	}
	paint.FillShape(gtx.Ops, color.NRGBA{A: 110}, clip.Rect{Max: gtx.Constraints.Max}.Op())
	layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X, gtx.Constraints.Max.X = gtx.Dp(408), gtx.Dp(408)
		return u.pollComposeCard(gtx)
	})
}

func (u *UI) pollComposeCard(gtx layout.Context) layout.Dimensions {
	white := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	field := func(gtx layout.Context, ed *widget.Editor, hint string) layout.Dimensions {
		macro := op.Record(gtx.Ops)
		dims := layout.Inset{Top: unit.Dp(9), Bottom: unit.Dp(9), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			e := material.Editor(u.th, ed, hint)
			e.Color, e.HintColor, e.TextSize = u.t.Text, u.t.Text2, unit.Sp(14.5)
			return e.Layout(gtx)
		})
		call := macro.Stop()
		r := gtx.Dp(8)
		paint.FillShape(gtx.Ops, u.t.SearchBg, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
		call.Add(gtx.Ops)
		return dims
	}
	macro := op.Record(gtx.Ops)
	dims := layout.UniformInset(unit.Dp(16)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		children := []layout.FlexChild{
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				l := material.Label(u.th, 17, "Buat polling")
				l.Color, l.Font.Weight = u.t.Text, font.SemiBold
				return l.Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { return field(gtx, &u.pollQEd, "Pertanyaan") }),
			layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
		}
		for i := range u.pollOptEds {
			ed := &u.pollOptEds[i]
			children = append(children,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions { return field(gtx, ed, "Opsi") }),
				layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
			)
		}
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{Size: gtx.Constraints.Min} }),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					b := material.Button(u.th, &u.pollCancel, "Batal")
					b.Background, b.Color, b.CornerRadius, b.TextSize = u.t.Bg2, u.t.Text, unit.Dp(8), unit.Sp(14)
					return b.Layout(gtx)
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					b := material.Button(u.th, &u.pollCreate, "Buat")
					b.Background, b.Color, b.CornerRadius, b.TextSize = u.t.Accent, white, unit.Dp(8), unit.Sp(14)
					return b.Layout(gtx)
				}),
			)
		}))
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
	})
	call := macro.Stop()
	r := gtx.Dp(12)
	paint.FillShape(gtx.Ops, u.t.SidebarBg, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
	call.Add(gtx.Ops)
	return dims
}

// fwdCtl membangun controller modal teruskan dari daftar chat nyata.
func (u *UI) fwdCtl() *FwdCtl {
	rows := make([]mvRow, len(u.chats))
	sel := make([]bool, len(u.chats))
	n := 0
	for i, c := range u.chats {
		rows[i] = mvRow{name: c.Name, sub: c.Preview}
		if u.fwdSel[c.ID] {
			sel[i] = true
			n++
		}
	}
	if len(u.fwdClicks) < len(u.chats) {
		u.fwdClicks = make([]widget.Clickable, len(u.chats))
	}
	return &FwdCtl{Rows: rows, Clicks: u.fwdClicks, Selected: sel, SelCount: n, Cancel: &u.fwdCancel, Send: &u.fwdSend}
}

// handleReaction memproses klik emoji di pemilih reaksi: bila ada target pesan →
// core.React; bila tidak (dibuka dari tombol emoji composer) → sisipkan ke editor.
func (u *UI) handleReaction(gtx layout.Context) {
	for i := range u.rpClicks {
		for u.rpClicks[i].Clicked(gtx) {
			glyph := RpEmoji()[i]
			if u.reactMsgID != "" {
				if u.core != nil {
					u.core.React(u.selected, u.reactMsgID, u.reactSender, glyph, u.reactFromMe)
				}
				u.reactMsgID = ""
			} else {
				u.editor.Insert(glyph) // sisip emoji ke pesan yg sedang diketik
			}
			u.overlay = ""
		}
	}
}

// replyDisplayName — nama yg ditampilkan di banner balas.
func (u *UI) replyDisplayName(m app.MessageDTO) string {
	if m.Dir == "out" {
		return "Anda"
	}
	if m.Sender != "" {
		return m.Sender
	}
	return u.selName
}

// isMediaType — pesan yg byte-nya bisa disimpan ke disk (tambah baris "Unduh").
func isMediaType(t string) bool {
	switch t {
	case "image", "video", "gif", "document", "voice", "audio", "ptt", "sticker":
		return true
	}
	return false
}

// ctxMenuView — menu aksi pesan (.menu): kartu bg + baris glyph+label klik. Untuk
// pesan media, tambahkan baris "Unduh" (simpan byte ke disk via OnSaveMedia).
func (u *UI) ctxMenuView(gtx layout.Context) layout.Dimensions {
	type ctxItem = struct{ icon, label, to string }
	items := append([]ctxItem{}, ctxMenu...)
	m := u.ctxMsg
	if !m.Revoked && strings.TrimSpace(m.Text) != "" { // salin teks (juga caption media)
		items = append(items, ctxItem{"copy", "Salin", ""})
	}
	switch {
	case m.Dir == "out" && !m.Revoked && (m.Type == "" || m.Type == "text"):
		// pesan teks sendiri → bisa di-Edit (SendEdit; jendela ~15mnt di engine).
		items = append(items, ctxItem{"editpen", "Edit", ""})
	case isMediaType(m.Type) && u.OnSaveMedia != nil:
		items = append(items, ctxItem{"download", "Unduh", ""})
	}
	if !m.Revoked { // sematkan / lepas sematan (PinMessage)
		pinLabel := "Sematkan"
		if u.isPinned(m.ID) {
			pinLabel = "Lepas sematan"
		}
		items = append(items, ctxItem{"pin", pinLabel, ""})
	}
	if !m.Revoked && (m.Type == "" || m.Type == "text") && strings.TrimSpace(m.Text) != "" {
		u.transMu.Lock()
		translated := u.transText[m.ID] != ""
		u.transMu.Unlock()
		lbl := "Terjemahkan"
		if translated {
			lbl = "Sembunyikan terjemahan"
		}
		items = append(items, ctxItem{"globe", lbl, ""}) // terjemah teks (toggle)
	}
	items = append(items, ctxItem{"message", "Pilih", ""}) // masuk mode pilih (multi)
	children := make([]layout.FlexChild, 0, len(items))
	for i := range items {
		i := i
		it := items[i]
		for u.ctxItems[i].Clicked(gtx) {
			u.doCtxAction(gtx, it.label) // jalankan aksi engine bila ada
			u.overlay = it.to            // pindah ke popup tujuan ("" = tutup)
		}
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return u.ctxItems[i].Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				macro := op.Record(gtx.Ops)
				dims := layout.Inset{Top: unit.Dp(9), Bottom: unit.Dp(9), Left: unit.Dp(14), Right: unit.Dp(14)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = gtx.Constraints.Max.X
					dcol := u.t.Text
					if it.label == "Hapus" {
						dcol = color.NRGBA{R: 0xe3, G: 0x5d, B: 0x6a, A: 0xff}
					}
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions { return icon(gtx, it.icon, 18, dcol) }),
						layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							lbl := material.Label(u.th, 14.5, it.label)
							lbl.Color = dcol
							return lbl.Layout(gtx)
						}),
					)
				})
				call := macro.Stop()
				if u.ctxItems[i].Hovered() { // bg hover → jelas item mana yg ditunjuk
					paint.FillShape(gtx.Ops, u.t.Hover, clip.Rect{Max: dims.Size}.Op())
				}
				call.Add(gtx.Ops)
				return dims
			})
		}))
	}
	macro := op.Record(gtx.Ops)
	dims := layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
	call := macro.Stop()
	rr := gtx.Dp(10)
	paint.FillShape(gtx.Ops, u.t.Bg, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: rr, NE: rr, SE: rr, SW: rr}.Op(gtx.Ops))
	call.Add(gtx.Ops)
	return dims
}

// chatTag — tag pointer per baris chat (deteksi klik-kanan).
type chatTag int

// chatCtxAction = item menu aksi baris chat (ikon, label, aksi, danger).
type chatCtxAction struct {
	icon, label, action string
	danger              bool
}

// chatCtxActions menghasilkan item menu menurut status chat (pin/mute toggle).
func chatCtxActions(c app.ChatDTO) []chatCtxAction {
	pin, mute := "Sematkan", "Bisukan"
	if c.Pinned {
		pin = "Lepas sematan"
	}
	if c.Muted {
		mute = "Bunyikan"
	}
	return []chatCtxAction{
		{"pin", pin, "pin", false},
		{"mute", mute, "mute", false},
		{"star", "Pesan berbintang", "starred", false},
		{"archive", "Arsipkan", "archive", false},
		{"message", "Tandai belum dibaca", "unread", false},
		{"trash", "Hapus chat", "delete", true},
	}
}

// doChatAction menjalankan aksi baris chat terhadap engine + refresh bila perlu.
func (u *UI) doChatAction(action string, c app.ChatDTO) {
	if action == "starred" { // buka panel berbintang (lintas chat) — bukan aksi engine
		u.view = "starred"
		u.starAt = time.Time{} // paksa muat ulang
		return
	}
	if u.core == nil {
		return
	}
	switch action {
	case "pin":
		u.core.Pin(c.ID, !c.Pinned)
	case "mute":
		u.core.Mute(c.ID, !c.Muted)
	case "archive":
		u.core.Archive(c.ID, true)
	case "unread":
		u.core.MarkUnread(c.ID, true)
	case "delete":
		u.core.DeleteChat(c.ID)
	}
	u.chats = u.core.GetChats()
}

// chatCtxView — menu aksi baris chat (klik-kanan): kartu + baris glyph+label.
func (u *UI) chatCtxView(gtx layout.Context) layout.Dimensions {
	if u.chatCtxChat.ID == "" {
		u.overlay = ""
		return layout.Dimensions{}
	}
	c := u.chatCtxChat // snapshot saat menu dibuka
	items := chatCtxActions(c)
	children := make([]layout.FlexChild, 0, len(items))
	for i := range items {
		i := i
		it := items[i]
		for u.chatCtxItems[i].Clicked(gtx) {
			u.doChatAction(it.action, c)
			u.overlay = ""
		}
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return u.chatCtxItems[i].Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(9), Bottom: unit.Dp(9), Left: unit.Dp(14), Right: unit.Dp(14)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = gtx.Constraints.Max.X
					dcol := u.t.Text
					if it.danger {
						dcol = color.NRGBA{R: 0xe3, G: 0x5d, B: 0x6a, A: 0xff}
					}
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions { return icon(gtx, it.icon, 18, dcol) }),
						layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							lbl := material.Label(u.th, 14.5, it.label)
							lbl.Color = dcol
							return lbl.Layout(gtx)
						}),
					)
				})
			})
		}))
	}
	macro := op.Record(gtx.Ops)
	dims := layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
	call := macro.Stop()
	rr := gtx.Dp(10)
	paint.FillShape(gtx.Ops, u.t.Bg, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: rr, NE: rr, SE: rr, SW: rr}.Op(gtx.Ops))
	call.Add(gtx.Ops)
	return dims
}

// waLogo — logo aplikasi WhatsLite (aset SVG penuh-warna: kotak hijau + bubble).
func (u *UI) waLogo(gtx layout.Context, dp int) layout.Dimensions {
	d := gtx.Dp(unit.Dp(dp))
	if iop, ok := logoOp(d); ok {
		iop.Add(gtx.Ops)
		paint.PaintOp{}.Add(gtx.Ops)
	}
	return layout.Dimensions{Size: image.Pt(d, d)}
}

// ---- rail (nav kiri, tombol klik → ganti view) ----
// titleBar — bilah judul custom (CSD Wayland), tinggi 34: area drag (ActionMove,
// ditangani compositor) + logo/judul kiri + tombol minimize/maximize/close kanan.
// Aksi window jalan hanya bila OnWinAction di-set (cmd/whatslite-gio); pada
// gio-shot/X11 titlebar tetap digambar tanpa efek.
func (u *UI) titleBar(gtx layout.Context) layout.Dimensions {
	h := gtx.Dp(34)
	w := gtx.Constraints.Max.X
	sz := image.Pt(w, h)
	paint.FillShape(gtx.Ops, u.t.RailBg, clip.Rect{Max: sz}.Op())
	paint.FillShape(gtx.Ops, u.t.Divider, clip.Rect{Min: image.Pt(0, h-1), Max: sz}.Op())

	for u.winMin.Clicked(gtx) {
		if u.OnWinAction != nil {
			u.OnWinAction("minimize")
		}
	}
	for u.winMax.Clicked(gtx) {
		u.winMaxed = !u.winMaxed
		if u.OnWinAction != nil {
			if u.winMaxed {
				u.OnWinAction("maximize")
			} else {
				u.OnWinAction("unmaximize")
			}
		}
	}
	for u.winClose.Clicked(gtx) {
		if u.OnWinAction != nil {
			u.OnWinAction("close")
		}
	}

	bw := gtx.Dp(46)
	btnsX := w - 3*bw // tombol window dipatok mutlak di kanan
	if btnsX < 0 {
		btnsX = 0
	}
	// area drag = bagian kiri (di luar tombol) → ActionMove (geser jendela).
	area := clip.Rect{Max: image.Pt(btnsX, h)}.Push(gtx.Ops)
	system.ActionInputOp(system.ActionMove).Add(gtx.Ops)
	area.Pop()

	// kiri: ikon + judul. Min.Y=0 → anak setinggi natural (logo & label sejajar
	// satu sama lain). Grup lalu digeser ke tengah vertikal bar (offset (h-20)/2).
	logoD := gtx.Dp(20)
	yoff := (h - logoD) / 2
	if yoff < 0 {
		yoff = 0
	}
	lgtx := gtx
	lgtx.Constraints.Min, lgtx.Constraints.Max = image.Pt(0, 0), image.Pt(btnsX, h)
	lo := op.Offset(image.Pt(0, yoff)).Push(gtx.Ops)
	layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(lgtx,
		layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return u.waLogo(gtx, 20) }),
		layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			lbl := material.Label(u.th, 13, "WhatsLite")
			lbl.Color = u.t.Text
			lbl.Font.Weight = font.Medium
			lbl.MaxLines = 1
			return lbl.Layout(gtx)
		}),
	)
	lo.Pop()

	// kanan: tiga tombol dipatok mutlak (Flexed tak melebar andal di Rigid vertikal).
	bgtx := gtx
	bgtx.Constraints.Min, bgtx.Constraints.Max = image.Pt(bw, h), image.Pt(bw, h)
	for i, b := range []struct {
		c    *widget.Clickable
		kind string
	}{{&u.winMin, "min"}, {&u.winMax, "max"}, {&u.winClose, "close"}} {
		off := op.Offset(image.Pt(btnsX+i*bw, 0)).Push(gtx.Ops)
		u.winBtn(bgtx, b.c, b.kind, h, bw)
		off.Pop()
	}
	return layout.Dimensions{Size: sz}
}

// winBtn — tombol window bw×h dgn glyph (min: garis, max: kotak, close: ✕). Hover:
// abu (close → merah, glyph putih). Digambar manual agar tajam di titlebar tipis.
func (u *UI) winBtn(gtx layout.Context, c *widget.Clickable, kind string, h, bw int) layout.Dimensions {
	sz := image.Pt(bw, h)
	return c.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min, gtx.Constraints.Max = sz, sz
		col := u.t.Text2
		if c.Hovered() {
			bg := u.t.Hover
			if kind == "close" {
				bg = color.NRGBA{R: 0xe0, G: 0x3b, B: 0x3b, A: 0xff}
				col = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
			}
			paint.FillShape(gtx.Ops, bg, clip.Rect{Max: sz}.Op())
		}
		cx, cy := bw/2, h/2
		s := gtx.Dp(5)
		t := gtx.Dp(1)
		if t < 1 {
			t = 1
		}
		switch kind {
		case "min":
			paint.FillShape(gtx.Ops, col, clip.Rect{Min: image.Pt(cx-s, cy), Max: image.Pt(cx+s, cy+t)}.Op())
		case "max":
			r := image.Rect(cx-s, cy-s, cx+s, cy+s)
			paint.FillShape(gtx.Ops, col, clip.Rect{Min: r.Min, Max: image.Pt(r.Max.X, r.Min.Y+t)}.Op())
			paint.FillShape(gtx.Ops, col, clip.Rect{Min: image.Pt(r.Min.X, r.Max.Y-t), Max: r.Max}.Op())
			paint.FillShape(gtx.Ops, col, clip.Rect{Min: r.Min, Max: image.Pt(r.Min.X+t, r.Max.Y)}.Op())
			paint.FillShape(gtx.Ops, col, clip.Rect{Min: image.Pt(r.Max.X-t, r.Min.Y), Max: r.Max}.Op())
		case "close":
			off := op.Offset(image.Pt(cx-gtx.Dp(8), cy-gtx.Dp(8))).Push(gtx.Ops)
			icon(gtx, "close", 16, col)
			off.Pop()
		}
		return layout.Dimensions{Size: sz}
	})
}

func (u *UI) rail(gtx layout.Context) layout.Dimensions {
	w := gtx.Dp(56)
	sz := image.Pt(w, gtx.Constraints.Max.Y)
	// Section 1 (rail) warna SAMA dgn section 2 (sidebar); pemisah = border kanan.
	paint.FillShape(gtx.Ops, u.t.SidebarBg, clip.Rect{Max: sz}.Op())
	paint.FillShape(gtx.Ops, u.t.Line, clip.Rect{Min: image.Pt(w-1, 0), Max: sz}.Op())
	gtx.Constraints.Min.X, gtx.Constraints.Max.X = w, w

	// kelompok atas: Meta AI + nav (chats..contacts); settings (gerigi) + avatar
	// profil dipisah ke DASAR rail. railNav terakhir = "settings".
	last := len(railNav) - 1
	for u.railMetaC.Clicked(gtx) { // Meta AI (placeholder — belum ada backend)
	}
	top := []layout.FlexChild{
		layout.Rigid(layout.Spacer{Height: unit.Dp(14)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return u.railIconBtn(gtx, &u.railMetaC, "metaai", "Meta AI", false)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
	}
	for i := 0; i < last; i++ {
		i := i
		top = append(top, layout.Rigid(func(gtx layout.Context) layout.Dimensions { return u.railBtn(gtx, i) }))
		top = append(top, layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout))
	}

	for u.railProfileClick.Clicked(gtx) { // avatar profil → setelan profil
		u.view, u.setSub = "settings", "profile"
		u.profLoaded = false
	}

	layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx, top...)
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{Size: gtx.Constraints.Min} }),
		// dasar: gerigi setelan, avatar profil.
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return u.railBtn(gtx, last) }),
		layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return u.railProfile(gtx) }),
		layout.Rigid(layout.Spacer{Height: unit.Dp(14)}.Layout),
	)
	return layout.Dimensions{Size: sz}
}

// railProfile — avatar bulat 34 di dasar rail (foto profil sendiri bila ada, else
// inisial). Klik → setelan profil. Hover → tooltip "Profil".
func (u *UI) railProfile(gtx layout.Context) layout.Dimensions {
	name := "Saya"
	if u.profName != "" {
		name = u.profName
	}
	return u.railProfileClick.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		dims := u.avatar(gtx, name, u.selfJID, 34)
		if u.railProfileClick.Hovered() {
			tm := op.Record(gtx.Ops)
			u.railTooltip(gtx, dims.Size.Y, "Profil")
			op.Defer(gtx.Ops, tm.Stop())
		}
		return dims
	})
}

func (u *UI) railBtn(gtx layout.Context, i int) layout.Dimensions {
	nav := railNav[i]
	for u.railClicks[i].Clicked(gtx) {
		u.view = nav.view
	}
	dims := u.railIconBtn(gtx, &u.railClicks[i], nav.icon, nav.label, u.view == nav.view)
	if nav.view == "chats" && !u.fullscreenOverlay() { // badge belum-dibaca (jangan nembus overlay penuh-layar)
		if n := u.totalUnread(); n > 0 {
			tm := op.Record(gtx.Ops)
			u.railBadge(gtx, dims.Size.X, n)
			op.Defer(gtx.Ops, tm.Stop())
		}
	}
	return dims
}

// fullscreenOverlay — true bila overlay aktif menutup SELURUH layar (viewer status,
// composer, picker emoji status, lightbox) → jangan gambar elemen op.Defer (badge
// rail/tooltip) di atasnya.
func (u *UI) fullscreenOverlay() bool {
	switch u.overlay {
	case "statusview", "statuscompose", "statusemoji", "lightbox":
		return true
	}
	return false
}

// totalUnread — jumlah semua chat belum-dibaca (badge ikon Chats di rail).
func (u *UI) totalUnread() int {
	n := 0
	for i := range u.chats {
		n += u.chats[i].Badge
	}
	return n
}

// railBadge — pil kecil accent berisi jumlah, dipatok di pojok kanan-atas ikon rail.
func (u *UI) railBadge(gtx layout.Context, btnW, n int) layout.Dimensions {
	txt := itoa(n)
	if n > 99 {
		txt = "99+"
	}
	h := gtx.Dp(16)
	padX := gtx.Dp(4)
	white := color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	macro := op.Record(gtx.Ops)
	cg := gtx
	cg.Constraints = layout.Constraints{Max: image.Pt(gtx.Dp(60), h)}
	lbl := material.Label(u.th, 10, txt)
	lbl.Color = white
	ld := lbl.Layout(cg)
	call := macro.Stop()
	w := ld.Size.X + 2*padX
	if w < h {
		w = h
	}
	// pojok kanan-atas ikon (sedikit menjorok keluar).
	off := op.Offset(image.Pt(btnW-w+gtx.Dp(6), -gtx.Dp(3))).Push(gtx.Ops)
	r := h / 2
	paint.FillShape(gtx.Ops, u.t.Accent, clip.RRect{Rect: image.Rectangle{Max: image.Pt(w, h)}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
	lo := op.Offset(image.Pt((w-ld.Size.X)/2, (h-ld.Size.Y)/2)).Push(gtx.Ops)
	call.Add(gtx.Ops)
	lo.Pop()
	off.Pop()
	return layout.Dimensions{}
}

// railIconBtn — tombol ikon rail 44px: bg aktif/hover + ikon, plus tooltip saat
// hover (digambar di atas via op.Defer, supaya tak terpotong panel sebelah).
func (u *UI) railIconBtn(gtx layout.Context, c *widget.Clickable, ico, tip string, active bool) layout.Dimensions {
	return c.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		d := gtx.Dp(44)
		sz := image.Pt(d, d)
		rad := d / 2
		bg := color.NRGBA{}
		if active {
			bg = color.NRGBA{R: 0, G: 168, B: 132, A: 38}
			rad = gtx.Dp(14)
		} else if c.Hovered() {
			bg = u.t.Hover
		}
		if bg.A > 0 {
			paint.FillShape(gtx.Ops, bg, clip.RRect{Rect: image.Rectangle{Max: sz}, NW: rad, NE: rad, SE: rad, SW: rad}.Op(gtx.Ops))
		}
		gtx.Constraints.Min, gtx.Constraints.Max = sz, sz
		col := u.t.RailIco
		if active {
			col = u.t.Accent
		}
		layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return icon(gtx, ico, 24, col)
		})
		if c.Hovered() && tip != "" { // tooltip di kanan ikon, di atas segalanya
			tm := op.Record(gtx.Ops)
			u.railTooltip(gtx, d, tip)
			op.Defer(gtx.Ops, tm.Stop())
		}
		return layout.Dimensions{Size: sz}
	})
}

// railTooltip — kotak keterangan kecil di kanan tombol rail (paritas tooltip
// shadcn): bg Bg2 membulat + label, dipusatkan vertikal terhadap tombol btnH.
func (u *UI) railTooltip(gtx layout.Context, btnH int, txt string) {
	gap := gtx.Dp(8)
	// Tooltip = INVERS tema (kontras, bukan sewarna bg): bg=Text, teks=SidebarBg.
	tipBg, tipFg := u.t.Text, u.t.SidebarBg
	macro := op.Record(gtx.Ops)
	cg := gtx
	// tombol membatasi Max ke 44px → label ter-elipsis "S...". Beri ruang ukur.
	cg.Constraints = layout.Constraints{Max: image.Pt(gtx.Dp(240), gtx.Dp(48))}
	dims := layout.Inset{Top: unit.Dp(6), Bottom: unit.Dp(6), Left: unit.Dp(10), Right: unit.Dp(10)}.Layout(cg, func(gtx layout.Context) layout.Dimensions {
		l := material.Label(u.th, 13, txt)
		l.Color, l.MaxLines = tipFg, 1
		return l.Layout(gtx)
	})
	call := macro.Stop()
	// caret kiri (segitiga) menunjuk ke ikon — jelas tooltip milik tombol mana.
	bx, cy, cw := btnH+gap, btnH/2, gtx.Dp(5)
	var p clip.Path
	p.Begin(gtx.Ops)
	p.MoveTo(f32.Pt(float32(bx-cw), float32(cy)))
	p.LineTo(f32.Pt(float32(bx), float32(cy-cw)))
	p.LineTo(f32.Pt(float32(bx), float32(cy+cw)))
	p.Close()
	paint.FillShape(gtx.Ops, tipBg, clip.Outline{Path: p.End()}.Op())
	// kotak tooltip.
	off := op.Offset(image.Pt(bx, cy-dims.Size.Y/2)).Push(gtx.Ops)
	r := gtx.Dp(6)
	paint.FillShape(gtx.Ops, tipBg, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
	call.Add(gtx.Ops)
	off.Pop()
}

// ---- sidebar (dispatch per view: settings/calls pane, else daftar chat) ----
func (u *UI) sidebar(gtx layout.Context) layout.Dimensions {
	w := gtx.Dp(468)
	gtx.Constraints.Min.X, gtx.Constraints.Max.X = w, w
	gtx.Constraints.Min.Y = gtx.Constraints.Max.Y
	sz := image.Pt(w, gtx.Constraints.Max.Y)
	switch u.view {
	case "settings":
		u.handleSettings(gtx)
		kd, ret, lock := true, 90, false
		if u.core != nil {
			kd, ret, lock = u.core.GetKeepDeleted(), u.core.GetRetention(), u.core.HasAppPIN()
		}
		ctl := &SettingsCtl{
			Dark: u.dark, KeepDeleted: kd, Retention: ret, AppLock: lock, Clicks: u.setClicks[:],
			Sub: u.setSub, Back: &u.setBack, ProfileClick: &u.setProfileClick,
			Notifications: u.core == nil || u.core.NotificationsOn(),
			Language:      "id", LangClicks: u.langClicks[:],
		}
		if u.core != nil {
			ctl.Language = u.core.Language()
		}
		if u.setSub != "" && u.core != nil { // data sub-pane
			switch u.setSub {
			case "profile":
				p := u.core.GetProfile()
				ctl.ProfName, ctl.ProfAbout, ctl.ProfPhone = p.Name, p.About, p.Phone
				if !u.profLoaded { // isi editor sekali (agar bisa diketik)
					u.profNameEd.SetText(p.Name)
					u.profAboutEd.SetText(p.About)
					u.profLoaded = true
				}
				ctl.ProfNameEd, ctl.ProfAboutEd, ctl.ProfSave = &u.profNameEd, &u.profAboutEd, &u.profSave
				ctl.AboutClicks = u.aboutClicks[:]
				ctl.AboutToggle, ctl.AboutOpen = &u.aboutToggle, u.aboutOpen
				ctl.PhotoClick, ctl.Avatar, ctl.SelfJID = &u.profPhotoClick, u.avatar, u.selfJID
				for u.profPhotoClick.Clicked(gtx) { // ketuk avatar → pilih & unggah foto profil
					if u.OnSetPhoto != nil {
						u.OnSetPhoto()
					}
				}
			case "account":
				ctl.ProfPhone = u.core.GetProfile().Phone
			case "storage":
				s := u.core.GetStorageUsage()
				ctl.StoreDB, ctl.StoreMedia, ctl.StoreMsgs = s.DBBytes, s.MediaBytes, s.MsgCount
			case "privacy":
				pv := u.core.GetPrivacy()
				ctl.Privacy = pv
				ctl.PrivacyClicks = u.privacyClicks[:]
				for i := range privacyOrder { // ketuk baris → siklus all→contacts→none
					if u.privacyClicks[i].Clicked(gtx) {
						k := privacyOrder[i].key
						u.core.SetPrivacy(k, nextPrivacy(pv[k]))
					}
				}
			}
		}
		return SettingsView(gtx, u.th, u.t, ctl)
	case "calls":
		return SidePanesView(gtx, u.th, u.t, u.callRows())
	case "contacts":
		groups := u.contactGroups()
		u.handleContactsPane(gtx)
		for u.gcNewBtn.Clicked(gtx) { // "Grup baru" → modal buat grup
			u.overlay = "groupcreate"
		}
		for u.ctNewContactBtn.Clicked(gtx) { // "Kontak baru" → modal tambah kontak
			u.ncName.SetText("")
			u.ncPhone.SetText("")
			u.ncErr = ""
			u.overlay = "newcontact"
		}
		onCtx := func(idx int) { // klik-kanan baris → menu konteks kontak
			if idx >= 0 && idx < len(u.contactFlat) {
				u.cctContact = u.contactFlat[idx]
				u.overlay = "contactctx"
			}
		}
		return ContactsPaneView(gtx, u.th, u.t, groups, u.contactPaneClicks, &u.gcNewBtn, &u.contactList, u.avatar, &u.ctSearchEd, u.infoClicks, &u.ctNewContactBtn, onCtx)
	case "status":
		items := u.statusRows()
		u.handleStatus(gtx)
		return StatusPaneView(gtx, u.th, u.t, items, u.statusClicks, u.avatar, u.profName, u.selfJID, &u.statusList, &u.stMyClick)
	case "channels":
		rows := u.channelRows()
		u.handleChannels(gtx, rows)
		return ChannelsPaneView(gtx, u.th, u.t, rows, u.chnCtl(rows))
	case "communities":
		rows := u.communityRows()
		u.handleCommunities(gtx, rows)
		for u.comNewBtn.Clicked(gtx) { // "Komunitas baru" → modal buat grup (proxy; komunitas = kumpulan grup)
			u.overlay = "groupcreate"
		}
		return CommunitiesPaneView(gtx, u.th, u.t, u.comCtl(rows))
	case "search":
		return SearchView(gtx, u.th, u.t, u.searchCtl(gtx))
	case "starred":
		return StarredPaneView(gtx, u.th, u.t, u.starredCtl(gtx))
	}
	paint.FillShape(gtx.Ops, u.t.SidebarBg, clip.Rect{Max: sz}.Op())

	u.handleChatFilter(gtx)
	u.computeShown()
	u.handleNewChat(gtx)
	u.handleChatClicks(gtx) // klik baris chat diproses DI LUAR layout (ala kontak)
	cfQuery := strings.TrimSpace(u.searchEd.Text())
	inviteLink := inviteLinkOf(cfQuery) // query = tautan undangan grup?
	for u.searchMsgClick.Clicked(gtx) { // "Cari pesan" → pencarian isi pesan global
		u.svPrevView = "chats"
		u.svEd.SetText(cfQuery)
		u.svHits = u.svHits[:0]
		u.view = "search"
		gtx.Execute(key.FocusCmd{Tag: &u.svEd})
	}
	for u.joinClick.Clicked(gtx) { // tautan undangan → modal pratinjau + gabung
		u.joinLink, u.joinPreview, u.overlay = inviteLink, "", "joinlink"
		link := inviteLink
		go func() {
			if u.core != nil {
				if nm := u.core.PreviewGroupLink(link); nm != "" {
					u.joinPreview = nm
				}
			}
		}()
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return paneHead(gtx, u.th, u.t, w, "Chat")
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return u.searchBar(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return u.filterChips(gtx)
		}),
		// query = tautan undangan → tawarkan "Gabung grup lewat tautan".
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if inviteLink == "" {
				return layout.Dimensions{}
			}
			return u.joinClick.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Left: unit.Dp(14), Right: unit.Dp(14), Top: unit.Dp(8), Bottom: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = gtx.Constraints.Max.X
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions { return icon(gtx, "invitelink", 18, u.t.Accent) }),
						layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							l := material.Label(u.th, 14, "Gabung grup lewat tautan")
							l.Color, l.MaxLines = u.t.Accent, 1
							return l.Layout(gtx)
						}),
					)
				})
			})
		}),
		// query teks (≥2 char) → tawarkan cari ISI pesan (global, FTS) selain filter chat.
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if inviteLink != "" || len([]rune(cfQuery)) < 2 || phoneQuery(cfQuery) != "" {
				return layout.Dimensions{}
			}
			return u.searchMsgClick.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Left: unit.Dp(14), Right: unit.Dp(14), Top: unit.Dp(8), Bottom: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = gtx.Constraints.Max.X
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions { return icon(gtx, "search", 18, u.t.Accent) }),
						layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							l := material.Label(u.th, 14, "Cari pesan \""+cfQuery+"\"")
							l.Color, l.MaxLines = u.t.Accent, 1
							return l.Layout(gtx)
						}),
					)
				})
			})
		}),
		// query berupa nomor telepon → tawarkan "mulai chat baru".
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if ph := phoneQuery(u.searchEd.Text()); ph != "" {
				return u.newChatRow(gtx, ph)
			}
			return layout.Dimensions{}
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return material.List(u.th, &u.chatList).Layout(gtx, len(u.shown), func(gtx layout.Context, i int) layout.Dimensions {
				return u.chatRow(gtx, u.shown[i])
			})
		}),
	)
}

// inviteLinkOf — kembalikan tautan bila query berisi tautan undangan grup
// WhatsApp (chat.whatsapp.com/...), else "".
func inviteLinkOf(s string) string {
	s = strings.TrimSpace(s)
	if strings.Contains(s, "chat.whatsapp.com/") {
		return s
	}
	return ""
}

// phoneQuery — kembalikan digit nomor bila teks pencarian "mirip nomor" (≥8 digit,
// hanya digit/+/spasi/strip), atau "" bila bukan. Utk tawaran mulai-chat-baru.
func phoneQuery(s string) string {
	s = strings.TrimSpace(s)
	digits := make([]rune, 0, len(s))
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
			digits = append(digits, r)
		case r == '+' || r == ' ' || r == '-' || r == '(' || r == ')':
		default:
			return "" // ada huruf → bukan nomor
		}
	}
	if len(digits) < 8 {
		return ""
	}
	return string(digits)
}

// handleNewChat — klik baris "mulai chat baru" → buka chat dgn JID nomor itu.
func (u *UI) handleNewChat(gtx layout.Context) {
	for u.newChatClick.Clicked(gtx) {
		ph := phoneQuery(u.searchEd.Text())
		if ph == "" {
			continue
		}
		jid := ph + "@s.whatsapp.net"
		u.selected, u.selName, u.selGroup = jid, "+"+ph, false
		u.searchEd.SetText("")
		if u.core != nil {
			u.core.OpenChat(jid)
			u.messages = u.core.GetMessages(jid)
		}
		u.msgList.ScrollTo(len(u.messages))
	}
}

// newChatRow — baris tawaran "Mulai chat baru" dgn nomor (ikon newchat accent).
func (u *UI) newChatRow(gtx layout.Context, ph string) layout.Dimensions {
	return u.newChatClick.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(20), Right: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					d := gtx.Dp(40)
					sz := image.Pt(d, d)
					paint.FillShape(gtx.Ops, u.t.Accent, clip.Ellipse{Max: sz}.Op(gtx.Ops))
					gtx.Constraints.Min, gtx.Constraints.Max = sz, sz
					layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return icon(gtx, "message", 20, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
					})
					return layout.Dimensions{Size: sz}
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(13)}.Layout),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							l := material.Label(u.th, 15.5, "Mulai chat baru")
							l.Color, l.Font.Weight = u.t.Text, font.Medium
							return l.Layout(gtx)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							l := material.Label(u.th, 13.5, "+"+ph)
							l.Color = u.t.Accent
							return l.Layout(gtx)
						}),
					)
				}),
			)
		})
	})
}

// chatFilterLabels — chip filter daftar chat (paritas Filters.svelte).
var chatFilterLabels = []string{"Semua", "Belum dibaca", "Favorit", "Grup"}

// handleChatFilter memproses klik chip filter.
func (u *UI) handleChatFilter(gtx layout.Context) {
	for i := range u.filterClicks {
		for u.filterClicks[i].Clicked(gtx) {
			u.filterSel = i
		}
	}
}

// computeShown menyaring u.chats menurut filter aktif + teks pencarian → u.shown.
func (u *UI) computeShown() {
	q := strings.ToLower(strings.TrimSpace(u.searchEd.Text()))
	u.shown = u.shown[:0]
	for i, c := range u.chats {
		switch u.filterSel {
		case 1: // Belum dibaca
			if !c.Unread && c.Badge == 0 {
				continue
			}
		case 2: // Favorit (disematkan)
			if !c.Pinned {
				continue
			}
		case 3: // Grup
			if !c.Group {
				continue
			}
		}
		if q != "" && !strings.Contains(strings.ToLower(c.Name), q) &&
			!strings.Contains(strings.ToLower(c.Preview), q) {
			continue
		}
		u.shown = append(u.shown, i)
	}
}

// searchBar — input pencarian membulat (ikon + editor) di var(--search-bg).
func (u *UI) searchBar(gtx layout.Context) layout.Dimensions {
	return layout.Inset{Left: unit.Dp(12), Right: unit.Dp(12), Top: unit.Dp(8), Bottom: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		focused := gtx.Focused(&u.searchEd)
		ico := u.t.Text2
		if focused {
			ico = u.t.Accent // ikon ikut accent saat fokus (modern)
		}
		macro := op.Record(gtx.Ops)
		dims := layout.Inset{Top: unit.Dp(9), Bottom: unit.Dp(9), Left: unit.Dp(14), Right: unit.Dp(14)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return icon(gtx, "search", 18, ico)
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					e := material.Editor(u.th, &u.searchEd, "Cari atau mulai chat baru")
					e.TextSize = unit.Sp(14)
					e.Color = u.t.Text
					e.HintColor = u.t.Text2
					return e.Layout(gtx)
				}),
			)
		})
		call := macro.Stop()
		w := gtx.Constraints.Max.X
		sz := image.Pt(w, dims.Size.Y)
		r := dims.Size.Y / 2 // pil penuh (modern)
		// fokus → cincin accent tipis (focus ring), else bg polos.
		if focused {
			paint.FillShape(gtx.Ops, u.t.Accent, clip.RRect{Rect: image.Rectangle{Max: sz}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
			bw := gtx.Dp(2)
			in := image.Rectangle{Min: image.Pt(bw, bw), Max: image.Pt(w-bw, dims.Size.Y-bw)}
			ir := r - bw
			paint.FillShape(gtx.Ops, u.t.SearchBg, clip.RRect{Rect: in, NW: ir, NE: ir, SE: ir, SW: ir}.Op(gtx.Ops))
		} else {
			paint.FillShape(gtx.Ops, u.t.SearchBg, clip.RRect{Rect: image.Rectangle{Max: sz}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
		}
		call.Add(gtx.Ops)
		return layout.Dimensions{Size: sz}
	})
}

// filterChips — baris chip filter (aktif = bg accent lembut + teks accent).
func (u *UI) filterChips(gtx layout.Context) layout.Dimensions {
	return layout.Inset{Left: unit.Dp(12), Top: unit.Dp(2), Bottom: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		children := make([]layout.FlexChild, 0, len(chatFilterLabels)*2)
		for i := range chatFilterLabels {
			if i > 0 {
				children = append(children, layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout))
			}
			idx := i
			children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return u.filterChip(gtx, idx)
			}))
		}
		return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, children...)
	})
}

func (u *UI) filterChip(gtx layout.Context, i int) layout.Dimensions {
	active := u.filterSel == i
	// inaktif = pil abu (jelas TOMBOL, bukan teks statis); aktif = pil accent.
	txtCol := u.t.Text2
	chipBg := u.t.Bg2
	switch {
	case active:
		txtCol = u.t.Accent
		chipBg = color.NRGBA{R: 0x00, G: 0xa8, B: 0x84, A: 0x2e} // accent lembut
	case u.filterClicks[i].Hovered():
		chipBg = u.t.Hover
		txtCol = u.t.Text
	}
	return u.filterClicks[i].Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		macro := op.Record(gtx.Ops)
		dims := layout.Inset{Top: unit.Dp(7), Bottom: unit.Dp(7), Left: unit.Dp(14), Right: unit.Dp(14)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			lbl := material.Label(u.th, 13, chatFilterLabels[i])
			lbl.Color = txtCol
			return lbl.Layout(gtx)
		})
		call := macro.Stop()
		r := dims.Size.Y / 2
		if chipBg.A > 0 {
			paint.FillShape(gtx.Ops, chipBg, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
		}
		call.Add(gtx.Ops)
		return dims
	})
}

// ---- baris chat (.chat-row) ----
func (u *UI) chatRow(gtx layout.Context, i int) layout.Dimensions {
	c := u.chats[i]
	// row (clickable + konten) direkam → tag klik-kanan jadi INDUK (push clip +
	// event.Op + replay) supaya clickable di dalam dapat PRIMARY (buka chat) &
	// tag dapat SECONDARY (menu). Tag sebagai sibling/di-atas mencuri primary → klik mati.
	// row direkam → tag klik-kanan jadi INDUK (push clip + event.Op + replay) → klik
	// PRIMARY tetap ke baris (lewat handleChatClicks), SECONDARY → menu konteks.
	rowMacro := op.Record(gtx.Ops)
	dims := u.rowInner(gtx, i, c)
	rowCall := rowMacro.Stop()
	tag := chatTag(i)
	for {
		ev, ok := gtx.Event(pointer.Filter{Target: tag, Kinds: pointer.Press})
		if !ok {
			break
		}
		if pe, ok := ev.(pointer.Event); ok && pe.Buttons.Contain(pointer.ButtonSecondary) {
			u.chatCtxChat = c
			u.overlay = "chatctx"
		}
	}
	area := clip.Rect{Max: dims.Size}.Push(gtx.Ops)
	event.Op(gtx.Ops, tag)
	rowCall.Add(gtx.Ops)
	area.Pop()
	return dims
}

// groupMembersSummary — subtitle grup ala WhatsApp: daftar nama anggota dipisah
// koma ("Kamu, Andi, Budi, …"); fallback "N anggota". Sekali per buka grup.
func (u *UI) groupMembersSummary(jid string) string {
	if u.core == nil {
		return ""
	}
	gi := u.core.GetGroupInfo(jid)
	if gi == nil || len(gi.Participants) == 0 {
		return ""
	}
	names := make([]string, 0, len(gi.Participants))
	for _, p := range gi.Participants {
		nm := p.Name
		if u.selfJID != "" && jidUser(p.JID) == jidUser(u.selfJID) {
			nm = "Kamu"
		}
		if nm == "" {
			nm = jidUser(p.JID)
		}
		names = append(names, nm)
	}
	s := strings.Join(names, ", ")
	if len(s) > 120 { // terlalu panjang → ringkas ke jumlah
		return itoa(len(gi.Participants)) + " anggota"
	}
	return s
}

// handleChatClicks — proses klik baris chat DI LUAR layout (ala handleContactsPane).
// Memanggil Clicked() di dalam Clickable.Layout sendiri ternyata tak terpicu di
// dalam material.List → pindah ke sini (klik bisa, section 3 muncul).
func (u *UI) handleChatClicks(gtx layout.Context) {
	for i := range u.chats {
		if i >= len(u.clicks) {
			break
		}
		for u.clicks[i].Clicked(gtx) {
			c := u.chats[i]
			u.selected, u.selName, u.selGroup = c.ID, c.Name, c.Group
			u.view, u.openChannel = "chats", ""
			u.groupSub = ""
			if u.core != nil {
				u.core.OpenChat(c.ID)
				u.messages = u.core.GetMessages(c.ID)
				u.prefetchHistory(c.ID)
				if c.Group { // ringkasan anggota grup (nama2 → fallback "N anggota") utk subtitle
					u.groupSub = u.groupMembersSummary(c.ID)
				}
			}
			u.captureUnreadDivider(c.Badge)
			u.msgList.ScrollTo(len(u.messages))
		}
	}
}

// rowInner — isi baris chat (clickable buka-chat + konten + bg hover/aktif).
func (u *UI) rowInner(gtx layout.Context, i int, c app.ChatDTO) layout.Dimensions {
	return u.clicks[i].Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		// rekam konten dulu → tahu ukuran baris → gambar bg hover/aktif di BELAKANG.
		macro := op.Record(gtx.Ops)
		// modern (Telegram/Linear): baris lebih lega (vert 12), avatar 54, tanpa divider.
		dims := layout.Inset{Top: unit.Dp(12), Bottom: unit.Dp(12), Left: unit.Dp(8), Right: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return u.avatarPresence(gtx, c)
					}),
					layout.Rigid(layout.Spacer{Width: unit.Dp(14)}.Layout),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return u.rowLine(gtx, c.Name, c.Time, 16.5, u.t.Text, u.t.Text2)
							}),
							layout.Rigid(layout.Spacer{Height: unit.Dp(3)}.Layout),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return u.previewLine(gtx, c)
							}),
						)
					}),
				)
			})
		})
		call := macro.Stop()
		// bg: kartu MEMBULAT dgn margin SIMETRIS kiri-kanan (bukan kotak full yg
		// kepotong scrollbar di kanan saja). aktif=Selected > hover=Hover.
		bg := color.NRGBA{}
		active := c.ID == u.selected
		hov := u.clicks[i].Hovered()
		if active {
			bg = u.t.Selected
		} else if hov {
			bg = u.t.Hover
		}
		m := gtx.Dp(7)   // margin kiri+kanan (simetris)
		vy := gtx.Dp(3)  // margin atas+bawah kartu
		rr := gtx.Dp(12) // sudut membulat
		if bg.A > 0 {
			rect := image.Rectangle{Min: image.Pt(m, vy), Max: image.Pt(dims.Size.X-m, dims.Size.Y-vy)}
			paint.FillShape(gtx.Ops, bg, clip.RRect{Rect: rect, NW: rr, NE: rr, SE: rr, SW: rr}.Op(gtx.Ops))
		}
		call.Add(gtx.Ops)
		return dims
	})
}

// avatarPresence — avatar 54 chat list + titik presence (DM saja) di kanan-bawah.
// 3-keadaan SESUAI yg diketahui (bukan tebakan):
//
//	"online"  → hijau #28c840
//	"offline" → abu (last-seen terlihat → memang offline)
//	""        → TANPA titik (disembunyikan/privacy/reciprocity/belum ada data)
//
// Grup tak punya presence → tanpa titik. Tak ada "merah offline": WhatsApp pun tak
// menampilkannya & aturan reciprocity bisa menyembunyikan presence semua kontak.
func (u *UI) avatarPresence(gtx layout.Context, c app.ChatDTO) layout.Dimensions {
	av := u.avatar(gtx, c.Name, c.ID, 54)
	if c.Group {
		return av // grup: tak ada presence per-pengguna → tanpa titik
	}
	// DM selalu punya titik: hijau bila online, abu bila selainnya (offline/belum
	// diketahui). Tak pakai "tanpa-titik" — presence sering sparse → DM jadi polos.
	col := u.t.Text2 // abu = tak online (offline / belum ada data)
	if c.Presence == "online" {
		col = color.NRGBA{R: 0x28, G: 0xc8, B: 0x40, A: 0xff} // #28c840 online (hijau)
	}
	dot := gtx.Dp(14)
	bw := gtx.Dp(2)  // cincin border var(--bg) 2px
	off := gtx.Dp(1) // right:-1; bottom:-1
	x := av.Size.X - dot + off
	y := av.Size.Y - dot + off
	paint.FillShape(gtx.Ops, u.t.Bg, clip.Ellipse{Min: image.Pt(x, y), Max: image.Pt(x+dot, y+dot)}.Op(gtx.Ops))
	paint.FillShape(gtx.Ops, col, clip.Ellipse{Min: image.Pt(x+bw, y+bw), Max: image.Pt(x+dot-bw, y+dot-bw)}.Op(gtx.Ops))
	return av
}

func (u *UI) rowLine(gtx layout.Context, name, t string, sp unit.Sp, nameCol, timeCol color.NRGBA) layout.Dimensions {
	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			lbl := material.Label(u.th, sp, name)
			lbl.Color = nameCol
			lbl.MaxLines = 1
			lbl.Font.Weight = font.Medium
			return lbl.Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			lbl := material.Label(u.th, 12, t)
			lbl.Color = timeCol
			return lbl.Layout(gtx)
		}),
	)
}

func (u *UI) previewLine(gtx layout.Context, c app.ChatDTO) layout.Dimensions {
	typing := u.typingOf(c.ID) // mengetik → override preview (accent)
	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			txt, col := c.Preview, u.t.Text2
			if typing != "" {
				txt, col = typing, u.t.Accent
			}
			return u.previewRow(gtx, txt, 14, col, 1) // ikon SVG utk ringkasan media
		}),
		// indikator: bisu (mute) + sematkan (pin) + badge belum-dibaca.
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if !c.Muted {
				return layout.Dimensions{}
			}
			return layout.Inset{Left: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions { return icon(gtx, "mute", 16, u.t.Text2) })
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if !c.Pinned {
				return layout.Dimensions{}
			}
			return layout.Inset{Left: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions { return icon(gtx, "pin", 16, u.t.Text2) })
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if c.Badge <= 0 {
				return layout.Dimensions{}
			}
			// pil angka belum-dibaca (DM + grup); >999 → "999+". Bisu → abu, selain accent.
			return layout.Inset{Left: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return u.badge(gtx, c.Badge, c.Muted)
			})
		}),
	)
}

// badge — pil belum-dibaca hijau yang LEBARNYA mengikuti teks (bukan lingkaran
// statis), supaya angka 2-3 digit tak keluar dari background. >999 → "999+".
func (u *UI) badge(gtx layout.Context, n int, muted bool) layout.Dimensions {
	txt := itoa(n)
	if n > 999 {
		txt = "999+"
	}
	h := gtx.Dp(20)   // tinggi pil
	padX := gtx.Dp(6) // padding kiri-kanan utk multi-digit
	white := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	pill := u.t.Accent
	if muted { // chat bisu → pil abu (paritas WhatsApp)
		pill = u.t.Text2
	}
	// ukur label dulu (rekam) → tahu lebar teks.
	macro := op.Record(gtx.Ops)
	cgtx := gtx
	cgtx.Constraints.Min = image.Point{}
	lbl := material.Label(u.th, 11, txt)
	lbl.Color = white
	lblDims := lbl.Layout(cgtx)
	call := macro.Stop()
	w := lblDims.Size.X + 2*padX
	if w < h { // jaga bentuk lingkaran utk 1 digit
		w = h
	}
	r := h / 2
	paint.FillShape(gtx.Ops, pill, clip.RRect{Rect: image.Rectangle{Max: image.Pt(w, h)}, SE: r, SW: r, NW: r, NE: r}.Op(gtx.Ops))
	// pusatkan label di dalam pil.
	off := op.Offset(image.Pt((w-lblDims.Size.X)/2, (h-lblDims.Size.Y)/2)).Push(gtx.Ops)
	call.Add(gtx.Ops)
	off.Pop()
	return layout.Dimensions{Size: image.Pt(w, h)}
}

// ---- avatar (lingkaran warna + inisial) ----
// ensureAvatar memuat foto profil (engine AvatarBytes) sekali per jid di goroutine
// latar → decode → cache di u.photos[name]. Tak memblok thread UI; sekali gagal
// tetap ditandai agar tak refetch terus.
// ensureCommonGroups memuat grup bersama dgn kontak (async, cache + cooldown 30s).
// Hasil di u.commonGroups[jid] → dibaca infoData (DM).
func (u *UI) ensureCommonGroups(jid string) {
	if u.core == nil || jid == "" {
		return
	}
	u.commonMu.Lock()
	if _, ok := u.commonGroups[jid]; ok {
		u.commonMu.Unlock()
		return
	}
	if t, ok := u.commonTried[jid]; ok && time.Since(t) < 30*time.Second {
		u.commonMu.Unlock()
		return
	}
	u.commonTried[jid] = time.Now()
	u.commonMu.Unlock()
	go func() {
		gs := u.core.GetCommonGroups(jid)
		out := make([]InfoMember, 0, len(gs))
		for _, g := range gs {
			out = append(out, InfoMember{Name: g.Name, JID: g.JID})
		}
		u.commonMu.Lock()
		u.commonGroups[jid] = out
		u.commonMu.Unlock()
	}()
}

func (u *UI) ensureAvatar(name, jid string) {
	if u.core == nil || jid == "" {
		return
	}
	u.photoMu.Lock()
	if _, ok := u.photos[name]; ok { // sudah punya foto → selesai
		u.photoMu.Unlock()
		return
	}
	if t, ok := u.photoTried[jid]; ok && time.Since(t) < 20*time.Second {
		u.photoMu.Unlock() // baru dicoba → tunggu cooldown (retry utk channel/foto lambat)
		return
	}
	u.photoTried[jid] = time.Now()
	u.photoMu.Unlock()
	go func() {
		b := u.core.AvatarBytes(jid)
		img := decodeImage(b)
		if img == nil {
			return // gagal/kosong → cooldown lewat ensureAvatar berikutnya
		}
		op := paint.NewImageOp(img)
		u.photoMu.Lock()
		u.photos[name] = op
		u.photoMu.Unlock()
	}()
}

// reactionPills — chip reaksi di bawah bubble (emoji + jumlah bila >1). Pil Bg2
// membulat; milik-sendiri di-beri batas accent. Kosong bila tak ada reaksi.
func (u *UI) reactionPills(gtx layout.Context, m app.MessageDTO) layout.Dimensions {
	if len(m.Reactions) == 0 {
		return layout.Dimensions{}
	}
	return layout.Inset{Top: unit.Dp(3)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		children := make([]layout.FlexChild, 0, len(m.Reactions)*2)
		for i, rx := range m.Reactions {
			if i > 0 {
				children = append(children, layout.Rigid(layout.Spacer{Width: unit.Dp(4)}.Layout))
			}
			rx := rx
			children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				txt := rx.Emoji
				if rx.Count > 1 {
					txt = rx.Emoji + " " + itoa(rx.Count)
				}
				macro := op.Record(gtx.Ops)
				dims := layout.Inset{Top: unit.Dp(3), Bottom: unit.Dp(3), Left: unit.Dp(8), Right: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					lbl := material.Label(u.th, 13, txt)
					lbl.Color = u.t.Text
					return lbl.Layout(gtx)
				})
				call := macro.Stop()
				r := dims.Size.Y / 2
				border := u.t.Bg2
				if rx.Mine {
					border = u.t.Accent
				}
				paint.FillShape(gtx.Ops, border, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
				bw := gtx.Dp(1)
				inner := image.Rectangle{Min: image.Pt(bw, bw), Max: image.Pt(dims.Size.X-bw, dims.Size.Y-bw)}
				paint.FillShape(gtx.Ops, u.t.Bg2, clip.RRect{Rect: inner, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
				call.Add(gtx.Ops)
				return dims
			}))
		}
		return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, children...)
	})
}

// locationBubble — kartu lokasi: kotak peta (placeholder Bg2 + ikon locpin) +
// baris alamat (locpin + m.Text). Tap → buka peta (follow-up).
func (u *UI) locationBubble(gtx layout.Context, m app.MessageDTO) layout.Dimensions {
	addr := m.Text
	if addr == "" {
		addr = "Lokasi"
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			w := gtx.Dp(220)
			h := gtx.Dp(120)
			box := image.Pt(w, h)
			r := gtx.Dp(10)
			cl := clip.RRect{Rect: image.Rectangle{Max: box}, NW: r, NE: r, SE: r, SW: r}.Push(gtx.Ops)
			// peta gaya: latar + "jalan" (garis terang) + 2 blok samar.
			paint.FillShape(gtx.Ops, u.t.Bg2, clip.Rect{Max: box}.Op())
			road := withAlpha(u.t.Text2, 0x33)
			rw := gtx.Dp(3)
			// jalan horizontal + vertikal + diagonal.
			paint.FillShape(gtx.Ops, road, clip.Rect{Min: image.Pt(0, h*2/5), Max: image.Pt(w, h*2/5+rw)}.Op())
			paint.FillShape(gtx.Ops, road, clip.Rect{Min: image.Pt(w*3/5, 0), Max: image.Pt(w*3/5+rw, h)}.Op())
			blk := withAlpha(u.t.Text2, 0x1f)
			paint.FillShape(gtx.Ops, blk, clip.Rect{Min: image.Pt(gtx.Dp(12), gtx.Dp(10)), Max: image.Pt(w*3/5-gtx.Dp(8), h*2/5-gtx.Dp(6))}.Op())
			paint.FillShape(gtx.Ops, blk, clip.Rect{Min: image.Pt(w*3/5+gtx.Dp(10), h*2/5+gtx.Dp(8)), Max: image.Pt(w-gtx.Dp(12), h-gtx.Dp(10))}.Op())
			cl.Pop()
			gtx.Constraints.Min, gtx.Constraints.Max = box, box
			layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions { return icon(gtx, "locpin", 32, u.t.Accent) })
			return layout.Dimensions{Size: box}
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions { return icon(gtx, "locpin", 15, u.t.Text2) }),
				layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					lbl := material.Label(u.th, 14, addr)
					lbl.Color = u.t.Text
					lbl.MaxLines = 1
					return lbl.Layout(gtx)
				}),
			)
		}),
	)
}

// contactBubble — kartu kontak: avatar + nama (m.Text) + tautan "Simpan" accent.
func (u *UI) contactBubble(gtx layout.Context, m app.MessageDTO) layout.Dimensions {
	name := m.Text
	if name == "" {
		name = "Kontak"
	}
	sub := m.Thumb // nomor telepon bila ada
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		// baris atas: avatar + nama + nomor.
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions { return u.avatar(gtx, name, "", 40) }),
				layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							lbl := material.Label(u.th, 14.5, name)
							lbl.Color, lbl.Font.Weight, lbl.MaxLines = u.t.Text, font.Medium, 1
							return lbl.Layout(gtx)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							if sub == "" {
								return layout.Dimensions{}
							}
							lbl := material.Label(u.th, 13, sub)
							lbl.Color, lbl.MaxLines = u.t.Text2, 1
							return lbl.Layout(gtx)
						}),
					)
				}),
			)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
		// divider tipis + tombol "Lihat kontak" full-width (ala WhatsApp).
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = gtx.Constraints.Max.X
					paint.FillShape(gtx.Ops, withAlpha(u.t.Text2, 0x40), clip.Rect{Max: image.Pt(gtx.Constraints.Max.X, gtx.Dp(1))}.Op())
					return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, gtx.Dp(1))}
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = gtx.Constraints.Max.X
					return layout.Inset{Top: unit.Dp(7), Bottom: unit.Dp(2)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							lbl := material.Label(u.th, 13.5, "Lihat kontak")
							lbl.Color, lbl.Font.Weight = u.t.Accent, font.Medium
							return lbl.Layout(gtx)
						})
					})
				}),
			)
		}),
	)
}

// docBubble — kartu dokumen: ikon docfile (kotak accent) + nama berkas + sub
// (PDF · ukuran · halaman). Tap → OnPlayVideo? tidak; dokumen dibuka via engine
// (follow-up). m.Text = nama berkas; DocMime/DocSize/DocPages utk sub.
func (u *UI) docBubble(gtx layout.Context, m app.MessageDTO) layout.Dimensions {
	name := m.Text
	if name == "" {
		name = "Dokumen"
	}
	var parts []string
	if ext := docExt(m.DocMime); ext != "" {
		parts = append(parts, ext)
	}
	if m.DocSize > 0 {
		parts = append(parts, fmtBytes(m.DocSize))
	}
	if m.DocPages > 0 {
		parts = append(parts, itoa(m.DocPages)+" hal")
	}
	sub := strings.Join(parts, " · ")
	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			d := gtx.Dp(40)
			sz := image.Pt(d, d)
			r := gtx.Dp(8)
			paint.FillShape(gtx.Ops, withAlpha(u.t.Accent, 0x33), clip.RRect{Rect: image.Rectangle{Max: sz}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
			gtx.Constraints.Min, gtx.Constraints.Max = sz, sz
			layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions { return icon(gtx, "docfile", 22, u.t.Accent) })
			return layout.Dimensions{Size: sz}
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					lbl := material.Label(u.th, 14.5, name)
					lbl.Color = u.t.Text
					lbl.MaxLines = 1
					return lbl.Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if sub == "" {
						return layout.Dimensions{}
					}
					return layout.Inset{Top: unit.Dp(2)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						lbl := material.Label(u.th, 12.5, sub)
						lbl.Color = u.t.Text2
						return lbl.Layout(gtx)
					})
				}),
			)
		}),
	)
}

// voiceBubble — pesan suara: tombol play + waveform (batang) + durasi. Tap pada
// bubble memutar (OnPlayVoice di bubble()).
func (u *UI) voiceBubble(gtx layout.Context, m app.MessageDTO) layout.Dimensions {
	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return u.playCircle(gtx, 34) }),
		layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return u.waveform(gtx, 0.35) }),
		layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			dur := m.Text
			if dur == "" {
				dur = "0:08"
			}
			lbl := material.Label(u.th, 11, dur)
			lbl.Color = u.t.Text2
			return lbl.Layout(gtx)
		}),
	)
}

// playCircle — tombol play bundar accent (lingkaran isi + segitiga putih).
func (u *UI) playCircle(gtx layout.Context, dp int) layout.Dimensions {
	d := gtx.Dp(unit.Dp(dp))
	sz := image.Pt(d, d)
	paint.FillShape(gtx.Ops, u.t.Accent, clip.Ellipse{Max: sz}.Op(gtx.Ops))
	gtx.Constraints.Min, gtx.Constraints.Max = sz, sz
	layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return icon(gtx, "play", dp*3/5, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
	})
	return layout.Dimensions{Size: sz}
}

// waveform — batang suara dgn PROGRESS: bagian terputar (prog 0..1) = accent,
// sisanya = text2. Titik scrubber di batas. Lebar mengisi (Flexed).
func (u *UI) waveform(gtx layout.Context, prog float32) layout.Dimensions {
	heights := []int{6, 11, 8, 14, 9, 16, 7, 12, 10, 15, 6, 13, 8, 11, 7, 13, 9, 12, 6, 10}
	gap := gtx.Dp(3)
	bw := gtx.Dp(2)
	maxH := gtx.Dp(20)
	w := gtx.Constraints.Max.X
	if w <= 0 {
		w = len(heights) * (bw + gap)
	}
	n := len(heights)
	step := w / n
	if step < bw+1 {
		step = bw + 1
	}
	playedX := int(float32(w) * prog)
	for i, h := range heights {
		hp := gtx.Dp(unit.Dp(h))
		x := i * step
		if x+bw > w {
			break
		}
		y := (maxH - hp) / 2
		col := u.t.Text2
		if x <= playedX {
			col = u.t.Accent
		}
		paint.FillShape(gtx.Ops, col, clip.RRect{Rect: image.Rectangle{Min: image.Pt(x, y), Max: image.Pt(x+bw, y+hp)}, NW: bw / 2, NE: bw / 2, SE: bw / 2, SW: bw / 2}.Op(gtx.Ops))
	}
	// scrubber (titik) di posisi progress.
	dot := gtx.Dp(7)
	dx := playedX - dot/2
	if dx < 0 {
		dx = 0
	}
	dy := (maxH - dot) / 2
	paint.FillShape(gtx.Ops, u.t.Accent, clip.Ellipse{Min: image.Pt(dx, dy), Max: image.Pt(dx+dot, dy+dot)}.Op(gtx.Ops))
	return layout.Dimensions{Size: image.Pt(w, maxH)}
}

// stickerBubble — stiker: gambar ~128 tanpa gelembung (bg transparan di bubble()).
func (u *UI) stickerBubble(gtx layout.Context, m app.MessageDTO) layout.Dimensions {
	u.ensureMedia(u.selected, m.ID, "sticker")
	u.mediaMu.Lock()
	iop, ok := u.media[m.ID]
	u.mediaMu.Unlock()
	d := gtx.Dp(128)
	box := image.Pt(d, d)
	if ok {
		s := iop.Size()
		if s.X > 0 && s.Y > 0 && s.Y != s.X {
			box = image.Pt(d, d*s.Y/s.X) // jaga rasio
		}
		cl := clip.Rect{Max: box}.Push(gtx.Ops)
		drawImageFill(gtx.Ops, iop, box.X)
		cl.Pop()
		return layout.Dimensions{Size: box}
	}
	// fallback: kotak transparan + ikon stiker.
	gtx.Constraints.Min, gtx.Constraints.Max = box, box
	layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions { return icon(gtx, "sticker", 48, u.t.Text2) })
	return layout.Dimensions{Size: box}
}

// musicBubble — berkas musik/audio: art kotak + judul (nama berkas) + bar progress
// tipis + durasi. Beda dari voice (ptt) yg pakai waveform.
func (u *UI) musicBubble(gtx layout.Context, m app.MessageDTO) layout.Dimensions {
	title := m.Text
	if title == "" {
		title = "Audio"
	}
	dur := m.Thumb // durasi bila ada (mis. "3:24")
	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return u.playCircle(gtx, 40) }),
		layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					lbl := material.Label(u.th, 14, title)
					lbl.Color, lbl.MaxLines = u.t.Text, 1
					return lbl.Layout(gtx)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(6)}.Layout),
				// bar progress tipis (track + isi accent 30%).
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					w := gtx.Constraints.Max.X
					h := gtx.Dp(3)
					paint.FillShape(gtx.Ops, withAlpha(u.t.Text2, 0x60), clip.RRect{Rect: image.Rectangle{Max: image.Pt(w, h)}, NW: h / 2, NE: h / 2, SE: h / 2, SW: h / 2}.Op(gtx.Ops))
					pw := w * 3 / 10
					paint.FillShape(gtx.Ops, u.t.Accent, clip.RRect{Rect: image.Rectangle{Max: image.Pt(pw, h)}, NW: h / 2, NE: h / 2, SE: h / 2, SW: h / 2}.Op(gtx.Ops))
					return layout.Dimensions{Size: image.Pt(w, h)}
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(5)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					d := dur
					if d == "" {
						d = "3:24"
					}
					lbl := material.Label(u.th, 11, d)
					lbl.Color = u.t.Text2
					return lbl.Layout(gtx)
				}),
			)
		}),
	)
}

// fmtBytes — ukuran berkas ringkas.
func fmtBytes(n int64) string {
	switch {
	case n >= 1<<20:
		return itoa(int(n>>20)) + " MB"
	case n >= 1<<10:
		return itoa(int(n>>10)) + " KB"
	default:
		return itoa(int(n)) + " B"
	}
}

// docExt — label jenis dari mime ("application/pdf"→"PDF").
func docExt(mime string) string {
	// urut: spreadsheet & word DULU sebelum "document" generik — sebab mime OOXML
	// (xlsx/docx) sama-sama mengandung "officedocument".
	switch {
	case mime == "":
		return ""
	case strings.Contains(mime, "pdf"):
		return "PDF"
	case strings.Contains(mime, "sheet") || strings.Contains(mime, "excel"):
		return "XLS"
	case strings.Contains(mime, "word"):
		return "DOC"
	case strings.Contains(mime, "zip"):
		return "ZIP"
	}
	return "FILE"
}

// withAlpha — warna dgn alpha berbeda.
func withAlpha(c color.NRGBA, a uint8) color.NRGBA { c.A = a; return c }

// mentionText — render teks pesan dgn FORMAT WhatsApp (*tebal* _miring_ ~coret~
// ```mono```), URL biru, @mention accent. richtext → wrap benar.
func (u *UI) mentionText(gtx layout.Context, text string, mentions []app.MentionDTO) layout.Dimensions {
	return u.formattedText(gtx, text, mentions, unit.Sp(15), "")
}

// formattedText — renderer teks kaya: parse marker WhatsApp → run bergaya, warnai
// URL (link, bisa diklik) + @mention (accent). id != "" → state interaktif per-pesan
// (deteksi klik URL → OnOpenURL). Dipakai bubble chat & post channel.
// clk — clickable per-id (lazy) utk peta tombol bubble (show-more / terjemah).
func (u *UI) clk(m map[string]*widget.Clickable, id string) *widget.Clickable {
	c := m[id]
	if c == nil {
		c = &widget.Clickable{}
		m[id] = c
	}
	return c
}

func (u *UI) formattedText(gtx layout.Context, text string, mentions []app.MentionDTO, size unit.Sp, id string) layout.Dimensions {
	base := rtSpanStyle{Size: size, Color: u.t.Text}
	spans := u.richSpans(text, mentions, base)
	st := &u.mentionState
	if id != "" {
		if u.linkStates == nil {
			u.linkStates = map[string]*rtInteractiveText{}
		}
		if u.linkStates[id] == nil {
			u.linkStates[id] = &rtInteractiveText{}
		}
		st = u.linkStates[id]
	}
	dims := rtText(st, u.th.Shaper, spans...).Layout(gtx)
	for { // klik URL → buka browser
		span, ev, ok := st.Update(gtx)
		if !ok {
			break
		}
		if ev.Type == rtClick && span != nil && u.OnOpenURL != nil {
			if url, _ := span.Get("url").(string); url != "" {
				u.OnOpenURL(url)
			}
		}
	}
	return dims
}

// fmtRun — potongan teks dgn gaya format WhatsApp aktif.
type fmtRun struct {
	text                       string
	bold, italic, strike, mono bool
}

// fmtMark — definisi satu marker format WhatsApp.
type fmtMark struct {
	ch                         string
	bold, italic, strike, mono bool
}

var fmtMarks = []fmtMark{
	{ch: "```", mono: true}, // monospace (triple backtick) — cek DULU sebelum '`'
	{ch: "*", bold: true},   // *tebal*
	{ch: "_", italic: true}, // _miring_
	{ch: "~", strike: true}, // ~coret~
}

// fmtRuns — pisah `s` jadi run bergaya menurut marker WhatsApp (rekursif → nested,
// mis. *_tebal miring_*). Marker dilucuti. Pasangan valid: pembuka tak diikuti
// spasi, penutup tak didahului spasi, isi tak kosong (ala WhatsApp).
func fmtRuns(s string, b, i, st, mo bool) []fmtRun {
	for pos := 0; pos < len(s); pos++ {
		for _, m := range fmtMarks {
			if !strings.HasPrefix(s[pos:], m.ch) {
				continue
			}
			oe := pos + len(m.ch)
			if oe >= len(s) || s[oe] == ' ' || s[oe] == '\n' { // pembuka diikuti spasi → bukan format
				continue
			}
			// cari penutup: marker sama, tak didahului spasi, isi non-kosong.
			j := -1
			for k := oe; k+len(m.ch) <= len(s); k++ {
				if strings.HasPrefix(s[k:], m.ch) && s[k-1] != ' ' && s[k-1] != '\n' && k > oe {
					j = k
					break
				}
			}
			if j < 0 {
				continue
			}
			var out []fmtRun
			out = append(out, fmtRuns(s[:pos], b, i, st, mo)...)
			out = append(out, fmtRuns(s[oe:j], b || m.bold, i || m.italic, st || m.strike, mo || m.mono)...)
			out = append(out, fmtRuns(s[j+len(m.ch):], b, i, st, mo)...)
			return out
		}
	}
	if s == "" {
		return nil
	}
	return []fmtRun{{text: s, bold: b, italic: i, strike: st, mono: mo}}
}

var reURL = regexp.MustCompile(`(?:https?://|www\.)[^\s]+`)

// richSpans — text → span richtext: gaya format (tebal/miring/coret/mono) +
// warna URL (accent) & @mention (accent/medium). Coret via U+0336 (richtext tak
// punya strikethrough).
func (u *UI) richSpans(text string, mentions []app.MentionDTO, base rtSpanStyle) []rtSpanStyle {
	var spans []rtSpanStyle
	for _, r := range fmtRuns(text, false, false, false, false) {
		st := base
		if r.bold {
			st.Font.Weight = font.Bold
		}
		if r.italic {
			st.Font.Style = font.Italic
		}
		if r.mono {
			st.Font.Typeface = "Go Mono"
		}
		acc := st
		acc.Color = u.t.Accent // @mention → accent
		link := st
		link.Color = u.t.Link // URL → biru
		// pisah run jadi span normal vs tautan(biru)/mention(accent).
		for _, sp := range coloredSpans(r.text, mentions, st, link, acc) {
			if r.strike {
				sp.Content = strikeThrough(sp.Content)
			}
			spans = append(spans, sp)
		}
	}
	if len(spans) == 0 {
		base.Content = text
		spans = []rtSpanStyle{base}
	}
	return spans
}

// coloredSpans — pisah `text` pada token accent (URL via reURL + "@Name" mention),
// warna accent; sisanya pakai `base`. Pertahankan Font dari base (gaya format).
func coloredSpans(text string, mentions []app.MentionDTO, base, linkSt, acc rtSpanStyle) []rtSpanStyle {
	type tok struct {
		s, e  int
		isURL bool
	}
	var toks []tok
	for _, m := range reURL.FindAllStringIndex(text, -1) {
		toks = append(toks, tok{m[0], m[1], true})
	}
	for _, mn := range mentions {
		if mn.Name == "" {
			continue
		}
		t := "@" + mn.Name
		for from := 0; ; {
			p := strings.Index(text[from:], t)
			if p < 0 {
				break
			}
			toks = append(toks, tok{from + p, from + p + len(t), false})
			from += p + len(t)
		}
	}
	if len(toks) == 0 {
		base.Content = text
		return []rtSpanStyle{base}
	}
	// urut token by start; abaikan tumpang-tindih.
	sort.Slice(toks, func(a, b int) bool { return toks[a].s < toks[b].s })
	var spans []rtSpanStyle
	i := 0
	for _, t := range toks {
		if t.s < i {
			continue // tumpang tindih → lewati
		}
		if t.s > i {
			s := base
			s.Content = text[i:t.s]
			spans = append(spans, s)
		}
		a := acc
		if t.isURL { // URL → biru + bisa diklik (buka browser) + metadata url
			a = linkSt
			a.Interactive = true
			a.Set("url", text[t.s:t.e])
		}
		a.Content = text[t.s:t.e]
		spans = append(spans, a)
		i = t.e
	}
	if i < len(text) {
		s := base
		s.Content = text[i:]
		spans = append(spans, s)
	}
	return spans
}

// strikeThrough — sisipkan U+0336 (combining long stroke overlay) tiap rune →
// efek coret (rtSpanStyle tak punya properti strikethrough).
func strikeThrough(s string) string {
	var b strings.Builder
	b.Grow(len(s) * 2)
	for _, r := range s {
		b.WriteRune(r)
		b.WriteRune('̶')
	}
	return b.String()
}

// mentionSpans — pisah `text` jadi span normal vs span accent pada token "@Name"
// (dari mentions). Pure → bisa diuji. Token paling-awal di tiap posisi yg dipilih.
func mentionSpans(text string, mentions []app.MentionDTO, base, acc rtSpanStyle) []rtSpanStyle {
	toks := make([]string, 0, len(mentions))
	for _, mn := range mentions {
		if mn.Name != "" {
			toks = append(toks, "@"+mn.Name)
		}
	}
	var spans []rtSpanStyle
	for i := 0; i < len(text); {
		bestPos, bestTok := -1, ""
		for _, tok := range toks {
			if p := strings.Index(text[i:], tok); p >= 0 && (bestPos < 0 || p < bestPos) {
				bestPos, bestTok = p, tok
			}
		}
		if bestPos < 0 {
			s := base
			s.Content = text[i:]
			spans = append(spans, s)
			break
		}
		if bestPos > 0 {
			s := base
			s.Content = text[i : i+bestPos]
			spans = append(spans, s)
		}
		s := acc
		s.Content = bestTok
		spans = append(spans, s)
		i += bestPos + len(bestTok)
	}
	return spans
}

// pollVoteEntry — hasil suara ter-cache + waktunya (TTL 2s).
type pollVoteEntry struct {
	v  app.PollVotesDTO
	at time.Time
}

// cachedPollVotes — GetPollVotes dgn TTL 2s per poll → tak query DB tiap frame
// (mis. saat scroll). Suara baru muncul ≤2s, cukup cepat utuk UI.
func (u *UI) cachedPollVotes(msgID string) app.PollVotesDTO {
	if e, ok := u.pollVoteCache[msgID]; ok && time.Since(e.at) < 2*time.Second {
		return e.v
	}
	v := u.core.GetPollVotes(msgID)
	u.pollVoteCache[msgID] = pollVoteEntry{v: v, at: time.Now()}
	return v
}

// pollBubble — kartu polling: ikon+pertanyaan (m.Text) + opsi bordered (m.Thumb =
// JSON []string). Tampilan (voting = follow-up). Paritas .poll-card.
func (u *UI) pollBubble(gtx layout.Context, m app.MessageDTO) layout.Dimensions {
	var opts []string
	if m.Thumb != "" {
		_ = json.Unmarshal([]byte(m.Thumb), &opts)
	}
	// hasil suara (counts/total) + clickable per opsi.
	var counts map[string]int
	total := 0
	if u.core != nil {
		v := u.cachedPollVotes(m.ID)
		counts, total = v.Counts, v.Total
	}
	clks := u.pollClicks[m.ID]
	if len(clks) < len(opts) {
		clks = make([]widget.Clickable, len(opts))
		u.pollClicks[m.ID] = clks
	}
	pollSender := m.SenderID
	if pollSender == "" {
		pollSender = m.Sender
	}
	children := make([]layout.FlexChild, 0, len(opts)*2+1)
	children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { return icon(gtx, "pollq", 16, u.t.Text2) }),
			layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				lbl := material.Label(u.th, 14.5, m.Text)
				lbl.Color = u.t.Text
				lbl.Font.Weight = font.SemiBold
				return lbl.Layout(gtx)
			}),
		)
	}))
	for i := range opts {
		o := opts[i]
		clk := &u.pollClicks[m.ID][i]
		for clk.Clicked(gtx) { // tap opsi → kirim suara
			if u.core != nil {
				u.core.VotePoll(u.selected, pollSender, m.ID, []string{o})
				delete(u.pollVoteCache, m.ID) // invalidasi → hitung baru tampil segera
			}
		}
		cnt := counts[o]
		children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout))
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return clk.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return u.pollOption(gtx, o, cnt, total)
			})
		}))
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
}

// pollOption — opsi polling: kotak bordered + radio + label + jumlah suara, dgn bar
// proporsi (cnt/total) berlatar accent lembut.
func (u *UI) pollOption(gtx layout.Context, label string, cnt, total int) layout.Dimensions {
	macro := op.Record(gtx.Ops)
	dims := layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				d := gtx.Dp(18)
				sz := image.Pt(d, d)
				paint.FillShape(gtx.Ops, u.t.Line, clip.Ellipse{Max: sz}.Op(gtx.Ops))
				bw := gtx.Dp(2)
				in := image.Rectangle{Min: image.Pt(bw, bw), Max: image.Pt(d-bw, d-bw)}
				paint.FillShape(gtx.Ops, u.t.InBg, clip.Ellipse{Min: in.Min, Max: in.Max}.Op(gtx.Ops))
				return layout.Dimensions{Size: sz}
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				lbl := material.Label(u.th, 14, label)
				lbl.Color = u.t.Text
				return lbl.Layout(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if total <= 0 {
					return layout.Dimensions{}
				}
				lbl := material.Label(u.th, 13, itoa(cnt))
				lbl.Color = u.t.Text2
				return lbl.Layout(gtx)
			}),
		)
	})
	call := macro.Stop()
	r := gtx.Dp(8)
	paint.FillShape(gtx.Ops, u.t.Line, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
	bw := gtx.Dp(1)
	inner := image.Rectangle{Min: image.Pt(bw, bw), Max: image.Pt(dims.Size.X-bw, dims.Size.Y-bw)}
	paint.FillShape(gtx.Ops, u.t.InBg, clip.RRect{Rect: inner, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
	if total > 0 && cnt > 0 { // bar proporsi accent lembut
		bw2 := (inner.Dx() * cnt) / total
		bar := image.Rectangle{Min: inner.Min, Max: image.Pt(inner.Min.X+bw2, inner.Max.Y)}
		paint.FillShape(gtx.Ops, withAlpha(u.t.Accent, 0x22), clip.RRect{Rect: bar, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
	}
	call.Add(gtx.Ops)
	return dims
}

// previewIcons — pemetaan emoji-awalan ringkasan media (storage/engine) → ikon SVG.
// Emoji VS16 (🖼️/🏷️ dll) sering jadi tofu; ganti dgn ikon garis biar rapi.
var previewIcons = []struct{ emoji, icon string }{
	{"🖼️", "media"}, {"🖼", "media"}, {"🎬", "video"}, {"🏷️", "sticker"}, {"🏷", "sticker"},
	{"🎤", "mic"}, {"🎵", "media"}, {"📄", "docfile"}, {"📍", "locpin"}, {"👤", "contacts"},
	{"🎞️", "gifb"}, {"🎞", "gifb"}, {"📊", "pollq"}, {"📷", "media"}, {"📸", "media"},
}

// mediaPreviewSplit — temukan emoji media di `s` → (ikonSVG, teksSebelum, teksSesudah).
// Emoji bisa di awal (kutipan: "🖼️ Foto") atau setelah "Nama: " (daftar chat). ""=tak ada.
func mediaPreviewSplit(s string) (ic, before, after string) {
	for _, p := range previewIcons {
		if i := strings.Index(s, p.emoji); i >= 0 {
			return p.icon, s[:i], strings.TrimSpace(s[i+len(p.emoji):])
		}
	}
	return "", s, ""
}

// previewRow — baris ringkasan: [sebelum] ikonSVG [sesudah] (ganti emoji tofu).
func (u *UI) previewRow(gtx layout.Context, text string, size unit.Sp, col color.NRGBA, maxLines int) layout.Dimensions {
	ic, before, after := mediaPreviewSplit(text)
	mk := func(s string) func(layout.Context) layout.Dimensions {
		return func(gtx layout.Context) layout.Dimensions {
			l := material.Label(u.th, size, s)
			l.Color, l.MaxLines = col, maxLines
			return l.Layout(gtx)
		}
	}
	if ic == "" {
		return mk(text)(gtx)
	}
	children := make([]layout.FlexChild, 0, 4)
	if before != "" {
		children = append(children, layout.Rigid(mk(before)))
	}
	children = append(children,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return icon(gtx, ic, int(size)+1, col) }),
		layout.Rigid(layout.Spacer{Width: unit.Dp(5)}.Layout),
		layout.Flexed(1, mk(after)),
	)
	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx, children...)
}

// quoteBlock — kutipan pesan yg dibalas, di dalam bubble (garis accent kiri + nama
// + teks), latar agak gelap. margin-bottom kecil sebelum isi.
func (u *UI) quoteBlock(gtx layout.Context, m app.MessageDTO, out bool) layout.Dimensions {
	name := m.QuoteName
	if name == "" {
		name = "Pesan"
	}
	return layout.Inset{Bottom: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		qcol := u.t.Text2
		if firstURL(m.QuoteText) != "" { // tautan di kutipan → biru
			qcol = u.t.Link
		}
		macro := op.Record(gtx.Ops)
		// HUG konten (jangan paksa lebar penuh) → balasan teks pendek = bubble pendek.
		dims := layout.Inset{Top: unit.Dp(5), Bottom: unit.Dp(5), Left: unit.Dp(10), Right: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					lbl := material.Label(u.th, 13, name)
					lbl.Color = u.t.Accent
					lbl.Font.Weight = font.Medium
					lbl.MaxLines = 1
					return lbl.Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return u.previewRow(gtx, m.QuoteText, 13, qcol, 1) // ikon SVG utk media + biru utk URL
				}),
			)
		})
		call := macro.Stop()
		r := gtx.Dp(5)
		// latar kutipan: sedikit kontras dari bubble.
		qbg := u.t.InBg
		if out {
			qbg = color.NRGBA{R: 0, G: 0, B: 0, A: 40}
		} else {
			qbg = color.NRGBA{R: 0, G: 0, B: 0, A: 30}
		}
		paint.FillShape(gtx.Ops, qbg, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
		paint.FillShape(gtx.Ops, u.t.Accent, clip.Rect{Max: image.Pt(gtx.Dp(3), dims.Size.Y)}.Op())
		call.Add(gtx.Ops)
		return dims
	})
}

// jumpToMessage menggulir ke pesan asal (ID) yg dikutip balasan & menyorotnya
// sesaat. Bila pesan belum dimuat (history lama), abaikan diam-diam.
func (u *UI) jumpToMessage(id string) {
	for i := range u.messages {
		if u.messages[i].ID == id {
			u.msgList.ScrollTo(i)
			u.hlMsg, u.hlAt = id, time.Now()
			return
		}
	}
}

// ensureMedia memuat byte media bubble (engine MediaBytes) sekali per msgID di
// goroutine → decode → cache u.media[id]. Tak memblok UI.
// ensureMedia memuat byte media penuh (async) → ImageOp di u.media[id]. kind
// menentukan fallback poster: image.Decode dulu; bila gagal & kind video/gif/
// sticker, ekstrak frame still via OnMediaPoster (ffmpeg) — GIF WA = mp4, stiker
// sering webp animasi (ANMF) yg tak bisa di-decode statis.
func (u *UI) ensureMedia(chat, id, kind string) {
	if u.core == nil || id == "" {
		return
	}
	u.mediaMu.Lock()
	if u.mediaTried[id] {
		u.mediaMu.Unlock()
		return
	}
	u.mediaTried[id] = true
	u.mediaMu.Unlock()
	go func() {
		b := u.core.MediaBytes(chat, id)
		img := decodeImage(b)
		if img == nil && u.OnMediaPoster != nil && len(b) > 0 {
			ext := ".mp4"
			if kind == "sticker" {
				ext = ".webp"
			}
			img = u.OnMediaPoster(b, ext) // GIF(mp4)/webp-animasi/video → poster frame
		}
		if img == nil {
			return
		}
		op := paint.NewImageOp(img)
		u.mediaMu.Lock()
		u.media[id] = op
		u.mediaMu.Unlock()
	}()
}

// mediaThumb — thumbnail media bubble (image/video/gif): kotak membulat 220px (rasio
// asli bila termuat, else 4:3 placeholder Bg2 + ikon), play-overlay utk video/gif,
// lalu caption m.Text bila ada. Tap di-tangani bubble (OnPlayVideo).
func (u *UI) mediaThumb(gtx layout.Context, m app.MessageDTO) layout.Dimensions {
	u.ensureMedia(u.selected, m.ID, m.Type)
	u.mediaMu.Lock()
	iop, ok := u.media[m.ID]
	u.mediaMu.Unlock()

	w := gtx.Dp(220)
	h := w * 3 / 4
	if ok {
		s := iop.Size()
		if s.X > 0 && s.Y > 0 {
			h = w * s.Y / s.X
			if max := gtx.Dp(300); h > max {
				h = max
			}
		}
	}
	box := image.Pt(w, h)
	r := gtx.Dp(10)

	thumb := func(gtx layout.Context) layout.Dimensions {
		if ok {
			cl := clip.RRect{Rect: image.Rectangle{Max: box}, NW: r, NE: r, SE: r, SW: r}.Push(gtx.Ops)
			drawImageFill(gtx.Ops, iop, w) // cover lebar; tinggi mengikuti
			_ = h
			cl.Pop()
		} else {
			paint.FillShape(gtx.Ops, u.t.Bg2, clip.RRect{Rect: image.Rectangle{Max: box}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
			ic := "wallpaperico"
			if m.Type != "image" {
				ic = "play"
			}
			gtx.Constraints.Min, gtx.Constraints.Max = box, box
			layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return icon(gtx, ic, 36, u.t.Text2)
			})
		}
		// play-overlay (video/gif): lingkaran gelap + segitiga putih di tengah.
		if m.Type == "video" || m.Type == "gif" {
			gtx.Constraints.Min, gtx.Constraints.Max = box, box
			layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				d := gtx.Dp(46)
				sz := image.Pt(d, d)
				paint.FillShape(gtx.Ops, color.NRGBA{A: 140}, clip.Ellipse{Max: sz}.Op(gtx.Ops))
				gtx.Constraints.Min, gtx.Constraints.Max = sz, sz
				layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return icon(gtx, "play", 22, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
				})
				return layout.Dimensions{Size: sz}
			})
		}
		// badge "GIF" pojok kiri-atas (chip gelap).
		if m.Type == "gif" {
			off := op.Offset(image.Pt(gtx.Dp(8), gtx.Dp(8))).Push(gtx.Ops)
			macro := op.Record(gtx.Ops)
			bd := layout.Inset{Top: unit.Dp(2), Bottom: unit.Dp(2), Left: unit.Dp(6), Right: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				lbl := material.Label(u.th, 11, "GIF")
				lbl.Color, lbl.Font.Weight = color.NRGBA{R: 255, G: 255, B: 255, A: 255}, font.Bold
				return lbl.Layout(gtx)
			})
			call := macro.Stop()
			rr := gtx.Dp(5)
			paint.FillShape(gtx.Ops, color.NRGBA{A: 150}, clip.RRect{Rect: image.Rectangle{Max: bd.Size}, NW: rr, NE: rr, SE: rr, SW: rr}.Op(gtx.Ops))
			call.Add(gtx.Ops)
			off.Pop()
		}
		// chip waktu + centang di kanan-bawah thumbnail (tanpa caption → overlay ala WA).
		if m.Text == "" {
			white := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
			cg := gtx
			cg.Constraints.Min = image.Point{} // play-overlay set Min=box → reset agar chip hug-content
			macro := op.Record(cg.Ops)
			ch := layout.Inset{Top: unit.Dp(3), Bottom: unit.Dp(3), Left: unit.Dp(7), Right: unit.Dp(7)}.Layout(cg, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						lbl := material.Label(u.th, 11, m.Time)
						lbl.Color = white
						return lbl.Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if m.Dir != "out" {
							return layout.Dimensions{}
						}
						return layout.Inset{Left: unit.Dp(3)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return icon(gtx, "checks", 14, white)
						})
					}),
				)
			})
			call := macro.Stop()
			mgn := gtx.Dp(6)
			off := op.Offset(image.Pt(box.X-ch.Size.X-mgn, box.Y-ch.Size.Y-mgn)).Push(gtx.Ops)
			rr := ch.Size.Y / 2
			paint.FillShape(gtx.Ops, color.NRGBA{A: 0x66}, clip.RRect{Rect: image.Rectangle{Max: ch.Size}, NW: rr, NE: rr, SE: rr, SW: rr}.Op(gtx.Ops))
			call.Add(gtx.Ops)
			off.Pop()
		}
		return layout.Dimensions{Size: box}
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(thumb),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if m.Text == "" {
				return layout.Dimensions{}
			}
			return layout.Inset{Top: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				lbl := material.Label(u.th, 15, m.Text)
				lbl.Color = u.t.Text
				return lbl.Layout(gtx)
			})
		}),
	)
}

// avatar — foto profil bulat (jid → AvatarBytes, di-cache); fallback inisial warna.
func (u *UI) avatar(gtx layout.Context, name, jid string, dp int) layout.Dimensions {
	d := gtx.Dp(unit.Dp(dp))
	sz := image.Pt(d, d)
	u.ensureAvatar(name, jid)
	// Foto in-memory (byte engine → ImageOp) di-mask bulat; else inisial.
	u.photoMu.Lock()
	ph, ok := u.photos[name]
	u.photoMu.Unlock()
	if ok {
		cl := clip.Ellipse{Max: sz}.Push(gtx.Ops)
		drawImageFill(gtx.Ops, ph, d)
		cl.Pop()
		return layout.Dimensions{Size: sz}
	}
	col := avatarColor(name)
	paint.FillShape(gtx.Ops, col, clip.Ellipse{Max: sz}.Op(gtx.Ops))
	gtx.Constraints.Min, gtx.Constraints.Max = sz, sz
	layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		lbl := material.Label(u.th, unit.Sp(float32(dp)*0.4), initial(name))
		lbl.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
		lbl.Font.Weight = font.Bold
		return lbl.Layout(gtx)
	})
	return layout.Dimensions{Size: sz}
}

// contactGroups membangun pane Kontak dari kontak nyata (core.GetContacts),
// dikelompokkan per huruf awal nama (urut). nil = mode demo.
func (u *UI) contactGroups() []cpGroup {
	if u.core == nil {
		return nil
	}
	q := strings.ToLower(strings.TrimSpace(u.ctSearchEd.Text()))
	if u.cgCache != nil && u.cgQuery == q && time.Since(u.cgAt) < time.Second {
		return u.cgCache // TTL: pertahankan contactFlat/clicks dari build terakhir
	}
	cs := u.core.GetContacts()
	sort.Slice(cs, func(i, j int) bool {
		return strings.ToLower(cs[i].Name) < strings.ToLower(cs[j].Name)
	})
	u.contactFlat = u.contactFlat[:0]
	var groups []cpGroup
	var cur *cpGroup
	for _, c := range cs {
		if c.Name == "" {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(c.Name), q) &&
			!strings.Contains(strings.ToLower(c.Phone), q) {
			continue // filter pencarian (nama / nomor)
		}
		letter := strings.ToUpper(initial(c.Name))
		if cur == nil || cur.letter != letter {
			groups = append(groups, cpGroup{letter: letter})
			cur = &groups[len(groups)-1]
		}
		idx := len(u.contactFlat)
		u.contactFlat = append(u.contactFlat, c)
		cur.items = append(cur.items, cpContact{name: c.Name, about: c.Phone, jid: c.JID, idx: idx})
	}
	if len(u.contactPaneClicks) < len(u.contactFlat) {
		u.contactPaneClicks = make([]widget.Clickable, len(u.contactFlat))
	}
	if len(u.infoClicks) < len(u.contactFlat) {
		u.infoClicks = make([]widget.Clickable, len(u.contactFlat))
	}
	u.cgCache, u.cgAt, u.cgQuery = groups, time.Now(), q
	return groups
}

// handleContactsPane — ketuk kontak → buka/mulai chat (pindah ke pane chats).
func (u *UI) handleContactsPane(gtx layout.Context) {
	for i := range u.contactFlat {
		if i >= len(u.contactPaneClicks) {
			break
		}
		for u.contactPaneClicks[i].Clicked(gtx) {
			c := u.contactFlat[i]
			u.selected, u.selName, u.selGroup = c.JID, c.Name, false
			u.view = "chats"
			if u.core != nil {
				u.core.OpenChat(c.JID)
				u.messages = u.core.GetMessages(c.JID)
				u.prefetchHistory(c.JID) // history tipis → backfill lama terurut
			}
			u.msgList.ScrollTo(len(u.messages))
		}
		if i < len(u.infoClicks) {
			for u.infoClicks[i].Clicked(gtx) { // ikon "i" → INTIP info kontak (tanpa buka percakapan)
				c := u.contactFlat[i]
				u.infoCJID, u.infoCName = c.JID, c.Name // sasaran info terpisah; selected tak disentuh
				u.overlay = "info"
			}
		}
	}
}

// channelRows membangun pane Saluran dari saluran nyata (core.GetChannels). nil = demo.
func (u *UI) channelRows() []chnChannel {
	if u.core == nil {
		return nil
	}
	if u.chnTab == 1 { // Jelajahi → direktori cari (query) atau rekomendasi (TTL 5s)
		q := strings.TrimSpace(u.chnSearchEd.Text())
		if u.chnExpCache != nil && u.chnExpQuery == q && time.Since(u.chnExpAt) < 5*time.Second {
			return u.chnExpCache
		}
		cs := u.core.GetRecommendedChannels(q)
		out := make([]chnChannel, 0, len(cs))
		for _, c := range cs {
			out = append(out, chnChannel{name: c.Name, subs: fmtSubs(c.Subscribers), jid: c.JID, follow: true, verified: c.Verified})
		}
		u.chnExpCache, u.chnExpAt, u.chnExpQuery = out, time.Now(), q
		return out
	}
	if u.chCache != nil && time.Since(u.chAt) < time.Second {
		return u.chCache
	}
	cs := u.core.GetChannels()
	out := make([]chnChannel, 0, len(cs))
	for _, c := range cs {
		out = append(out, chnChannel{name: c.Name, subs: fmtSubs(c.Subscribers), jid: c.JID, verified: c.Verified})
	}
	u.chCache, u.chAt = out, time.Now()
	return out
}

// channelMsgs — post channel terbuka (cache TTL 3s).
func (u *UI) channelMsgs() []app.ChannelMsgDTO {
	if u.core == nil || u.openChannel == "" {
		return nil
	}
	if u.chMsgsJID == u.openChannel && time.Since(u.chMsgsAt) < 3*time.Second {
		return u.chMsgsCache
	}
	u.chMsgsCache = u.core.GetChannelMessages(u.openChannel)
	u.chMsgsJID, u.chMsgsAt = u.openChannel, time.Now()
	return u.chMsgsCache
}

// channelReader — pembaca post channel (read-only) di section 3: header
// (avatar + nama) + daftar post (teks/thumb + waktu + views).
func (u *UI) channelReader(gtx layout.Context) layout.Dimensions {
	msgs := u.channelMsgs()
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		// header: avatar + nama channel.
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			h := gtx.Dp(60)
			sz := image.Pt(gtx.Constraints.Max.X, h)
			paint.FillShape(gtx.Ops, u.t.HeadBg, clip.Rect{Max: sz}.Op())
			paint.FillShape(gtx.Ops, u.t.Divider, clip.Rect{Min: image.Pt(0, h-gtx.Dp(1)), Max: sz}.Op())
			gtx.Constraints.Min, gtx.Constraints.Max = sz, sz
			return layout.Inset{Left: unit.Dp(16), Right: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions { return u.avatar(gtx, u.openChanName, u.openChannel, 42) }),
					layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										lbl := material.Label(u.th, 16, u.openChanName)
										lbl.Color, lbl.Font.Weight, lbl.MaxLines = u.t.Text, font.Medium, 1
										return lbl.Layout(gtx)
									}),
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										if !u.openChanVer {
											return layout.Dimensions{}
										}
										return layout.Inset{Left: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions { return icon(gtx, "verif", 15, u.t.Accent) })
									}),
								)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								if u.openChanSubs == "" {
									return layout.Dimensions{}
								}
								lbl := material.Label(u.th, 12.5, u.openChanSubs)
								lbl.Color, lbl.MaxLines = u.t.Text2, 1
								return lbl.Layout(gtx)
							}),
						)
					}),
				)
			})
		}),
		// daftar post (scroll).
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			if len(msgs) == 0 {
				return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					l := material.Label(u.th, 14, "Belum ada postingan")
					l.Color = u.t.Text2
					return l.Layout(gtx)
				})
			}
			u.chMsgList.Axis = layout.Vertical
			return material.List(u.th, &u.chMsgList).Layout(gtx, len(msgs), func(gtx layout.Context, i int) layout.Dimensions {
				return u.channelPost(gtx, msgs[i])
			})
		}),
	)
}

// channelPost — satu kartu post channel: teks + (waktu · N views).
func (u *UI) channelPost(gtx layout.Context, m app.ChannelMsgDTO) layout.Dimensions {
	return layout.Inset{Top: unit.Dp(6), Bottom: unit.Dp(6), Left: unit.Dp(14), Right: unit.Dp(60)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		macro := op.Record(gtx.Ops)
		dims := layout.Inset{Top: unit.Dp(9), Bottom: unit.Dp(9), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				// thumbnail media (image/video/sticker) dari m.Thumb (data-URI), bila ada.
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if m.Type != "image" && m.Type != "video" && m.Type != "sticker" {
						return layout.Dimensions{}
					}
					img := decodeImage(decodeDataURI(m.Thumb))
					if img == nil {
						return layout.Dimensions{}
					}
					iop := paint.NewImageOp(img)
					sz := iop.Size()
					w := gtx.Dp(220)
					h := w * 3 / 4
					if sz.X > 0 && sz.Y > 0 {
						h = w * sz.Y / sz.X
						if mx := gtx.Dp(280); h > mx {
							h = mx
						}
					}
					box := image.Pt(w, h)
					rr := gtx.Dp(8)
					cl := clip.RRect{Rect: image.Rectangle{Max: box}, NW: rr, NE: rr, SE: rr, SW: rr}.Push(gtx.Ops)
					drawImageFill(gtx.Ops, iop, w)
					cl.Pop()
					if m.Type == "video" { // play overlay
						gtx.Constraints.Min, gtx.Constraints.Max = box, box
						layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							d := gtx.Dp(44)
							ps := image.Pt(d, d)
							paint.FillShape(gtx.Ops, color.NRGBA{A: 0xaa}, clip.Ellipse{Max: ps}.Op(gtx.Ops))
							gtx.Constraints.Min, gtx.Constraints.Max = ps, ps
							layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return icon(gtx, "play", 24, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
							})
							return layout.Dimensions{Size: ps}
						})
					}
					return layout.Inset{Bottom: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{Size: box} })
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if m.Text == "" && m.Type != "" && m.Type != "text" {
						return layout.Dimensions{} // media tanpa caption → tak ada baris teks
					}
					return u.formattedText(gtx, m.Text, nil, unit.Sp(15), m.ID) // format + URL biru
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					meta := m.Time
					if m.Views > 0 {
						meta += "  ·  " + fmtSubs(m.Views) + " dilihat"
					}
					gtx.Constraints.Min.X = gtx.Constraints.Max.X
					return layout.E.Layout(gtx, func(gtx layout.Context) layout.Dimensions { // meta kanan-bawah (ala bubble)
						lbl := material.Label(u.th, 11.5, meta)
						lbl.Color = u.t.Text2
						return lbl.Layout(gtx)
					})
				}),
			)
		})
		call := macro.Stop()
		r := gtx.Dp(12)
		paint.FillShape(gtx.Ops, u.t.InBg, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
		call.Add(gtx.Ops)
		return dims
	})
}

// chnCtl — state interaktif pane channels (tab aktif + clickable tab/baris).
func (u *UI) chnCtl(rows []chnChannel) *ChnCtl {
	if len(u.chnRowClicks) < len(rows) {
		u.chnRowClicks = make([]widget.Clickable, len(rows))
	}
	if len(u.chnRowOpens) < len(rows) {
		u.chnRowOpens = make([]widget.Clickable, len(rows))
	}
	return &ChnCtl{Tabs: u.chnTabClicks[:], Active: u.chnTab, Rows: u.chnRowClicks, Opens: u.chnRowOpens, Search: &u.chnSearchEd, Av: u.avatar}
}

// handleChannels — proses klik tab (Diikuti/Jelajahi) + aksi baris (ikuti/unfollow).
func (u *UI) handleChannels(gtx layout.Context, rows []chnChannel) {
	for i := range u.chnTabClicks {
		for u.chnTabClicks[i].Clicked(gtx) {
			if u.chnTab != i {
				u.chnTab = i
				u.chCache, u.chnExpCache = nil, nil // muat ulang daftar tab baru
			}
		}
	}
	for i := range rows {
		if i >= len(u.chnRowClicks) {
			break
		}
		for u.chnRowClicks[i].Clicked(gtx) {
			if u.core == nil || rows[i].jid == "" {
				continue
			}
			if rows[i].follow {
				u.core.FollowChannelByJID(rows[i].jid)
			} else {
				u.core.UnfollowChannel(rows[i].jid)
			}
			u.chCache, u.chnExpCache = nil, nil
		}
		if i < len(u.chnRowOpens) { // tap baris diikuti → buka reader channel
			for u.chnRowOpens[i].Clicked(gtx) {
				if rows[i].jid != "" {
					u.openChannel, u.openChanName = rows[i].jid, rows[i].name
					u.openChanSubs, u.openChanVer = rows[i].subs, rows[i].verified
					u.chMsgsJID = "" // paksa muat post
				}
			}
		}
	}
}

// searchCtl menjalankan pencarian pesan global (FTS5 core.SearchMessages) dari
// query editor + bangun hit rows (klik → buka chat). Tombol kembali → view semula.
func (u *UI) searchCtl(gtx layout.Context) *SvCtl {
	for u.svBack.Clicked(gtx) {
		u.view = u.svPrevView
		if u.view == "" || u.view == "search" {
			u.view = "chats"
		}
	}
	q := strings.TrimSpace(u.svEd.Text())
	if u.core != nil && len(q) >= 2 {
		raw := u.core.SearchMessages(q, "")
		u.svHits = u.svHits[:0]
		for _, h := range raw {
			u.svHits = append(u.svHits, svHit{name: h.ChatName, text: h.Text, time: h.Time, jid: h.ChatJID})
		}
	} else {
		u.svHits = u.svHits[:0]
	}
	if len(u.svHitClicks) < len(u.svHits) {
		u.svHitClicks = make([]widget.Clickable, len(u.svHits))
	}
	for i := range u.svHits {
		if i >= len(u.svHitClicks) {
			break
		}
		for u.svHitClicks[i].Clicked(gtx) { // buka chat hit
			h := u.svHits[i]
			u.selected, u.selName, u.selGroup = h.jid, h.name, isGroupJIDStr(h.jid)
			u.view = "chats"
			if u.core != nil {
				u.core.OpenChat(h.jid)
				u.messages = u.core.GetMessages(h.jid)
			}
			u.msgList.ScrollTo(len(u.messages))
		}
	}
	return &SvCtl{Query: &u.svEd, Hits: u.svHits, HitClicks: u.svHitClicks, Back: &u.svBack}
}

// starredCtl membangun panel "Pesan berbintang" dari core.GetStarred (TTL 2s).
// Klik baris → buka chat di pesan tsb. Tombol kembali → daftar chat.
func (u *UI) starredCtl(gtx layout.Context) *StarredCtl {
	for u.starBack.Clicked(gtx) {
		u.view = "chats"
	}
	if u.core != nil && time.Since(u.starAt) > 2*time.Second {
		raw := u.core.GetStarred()
		u.starHits = u.starHits[:0]
		for _, h := range raw {
			u.starHits = append(u.starHits, svHit{name: h.ChatName, text: h.Text, time: h.Time, jid: h.ChatJID})
		}
		u.starAt = time.Now()
	}
	if len(u.starHitClicks) < len(u.starHits) {
		u.starHitClicks = make([]widget.Clickable, len(u.starHits))
	}
	for i := range u.starHits {
		if i >= len(u.starHitClicks) {
			break
		}
		for u.starHitClicks[i].Clicked(gtx) { // buka chat berbintang
			h := u.starHits[i]
			u.selected, u.selName, u.selGroup = h.jid, h.name, isGroupJIDStr(h.jid)
			u.view = "chats"
			if u.core != nil {
				u.core.OpenChat(h.jid)
				u.messages = u.core.GetMessages(h.jid)
			}
			u.msgList.ScrollTo(len(u.messages))
		}
	}
	return &StarredCtl{Hits: u.starHits, HitClicks: u.starHitClicks, Back: &u.starBack}
}

// emojiOnlyCount — jumlah emoji bila `s` HANYA berisi emoji (+ spasi/modifier),
// else 0. Dipakai memperbesar bubble emoji-saja (1-3) ala WhatsApp. Heuristik
// rentang Unicode emoji umum; karakter biasa apa pun → 0.
func emojiOnlyCount(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	count := 0
	prevZWJ := false
	// emoji dasar: tambah count KECUALI digabung ZWJ ke emoji sebelumnya (1 grafem).
	emoji := func() {
		if !prevZWJ {
			count++
		}
		prevZWJ = false
	}
	for _, r := range s {
		switch {
		case r == 0x200D: // ZWJ → emoji berikut bagian dari grafem yg sama
			prevZWJ = true
		case r == 0xFE0F || r == 0xFE0E: // variation selector — tak ubah status
		case r >= 0x1F3FB && r <= 0x1F3FF: // skin-tone modifier (bagian emoji sebelum)
		case r >= 0x1F1E6 && r <= 0x1F1FF: // regional indicator (bendera)
			emoji()
		case r >= 0x1F300 && r <= 0x1FAFF: // blok emoji utama
			emoji()
		case r >= 0x2600 && r <= 0x27BF: // simbol misc + dingbats
			emoji()
		case r >= 0x1F000 && r <= 0x1F0FF: // mahjong/domino/kartu
			emoji()
		case r == 0x2B50 || r == 0x2B55 || r == 0x2934 || r == 0x2935:
			emoji()
		case r == 0x203C || r == 0x2049 || (r >= 0x2190 && r <= 0x21AA):
			emoji()
		case unicode.IsSpace(r):
			prevZWJ = false
		default:
			return 0 // karakter biasa → bukan emoji-saja
		}
	}
	if count >= 1 && count <= 3 {
		return count
	}
	return 0
}

// captureUnreadDivider — saat buka chat, tandai pesan tempat divider "belum dibaca"
// digambar (sebelum OpenChat menandai-baca). Boundary ≈ pesan ke-(len-unread); tetap
// sampai pindah chat. unread<=0 → tak ada divider.
func (u *UI) captureUnreadDivider(unread int) {
	u.unreadDivID, u.unreadDivCount = "", 0
	if unread <= 0 {
		return
	}
	n := len(u.messages) - unread
	if n < 0 || n >= len(u.messages) {
		return
	}
	u.unreadDivID, u.unreadDivCount = u.messages[n].ID, unread
}

// pinnedMsgs — pesan tersemat chat aktif (GetPinned, TTL 2s + invalidate saat ganti chat).
func (u *UI) pinnedMsgs() []app.MessageDTO {
	if u.core == nil {
		return u.pinnedCache // demo: di-inject via SetPinnedDemo
	}
	if u.selected == "" {
		return nil
	}
	if u.pinnedChat != u.selected || time.Since(u.pinnedAt) > 2*time.Second {
		u.pinnedCache = u.core.GetPinned(u.selected)
		u.pinnedChat, u.pinnedAt = u.selected, time.Now()
	}
	return u.pinnedCache
}

// isPinned — true bila msgID tersemat di chat aktif.
func (u *UI) isPinned(id string) bool {
	for _, p := range u.pinnedMsgs() {
		if p.ID == id {
			return true
		}
	}
	return false
}

// ensureTranslate — terjemahkan teks pesan (async, sekali per msgID) ke bahasa app
// (id). Bila hasil == asli (sudah berbahasa id), tak disimpan (tak ada baris).
func (u *UI) ensureTranslate(id, text string) {
	if u.core == nil || strings.TrimSpace(text) == "" {
		return
	}
	u.transMu.Lock()
	if u.transTried[id] {
		u.transMu.Unlock()
		return
	}
	u.transTried[id] = true
	u.transMu.Unlock()
	go func() {
		out := u.core.Translate(text, "id")
		if out != "" && out != text {
			u.transMu.Lock()
			u.transText[id] = out
			u.transMu.Unlock()
		}
	}()
}

// translatedWidget — widget baris terjemahan ("Diterjemahkan" + teks) bila ada.
func (u *UI) translatedWidget(id string) layout.Widget {
	u.transMu.Lock()
	tr := u.transText[id]
	u.transMu.Unlock()
	if tr == "" {
		return nil
	}
	return func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				l := material.Label(u.th, 11, "Diterjemahkan")
				l.Color = u.t.Accent
				return l.Layout(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				l := material.Label(u.th, 15, tr)
				l.Color = u.t.Text
				return l.Layout(gtx)
			}),
		)
	}
}

// firstURL — URL http(s) pertama dalam teks (buang tanda baca ekor). "" bila tak ada.
func firstURL(s string) string {
	for _, f := range strings.Fields(s) {
		if strings.HasPrefix(f, "http://") || strings.HasPrefix(f, "https://") {
			return strings.TrimRight(f, ".,)!?:;\"'")
		}
	}
	return ""
}

// urlHost — host (domain) dari URL utk label kartu pratinjau.
func urlHost(u string) string {
	s := strings.TrimPrefix(strings.TrimPrefix(u, "https://"), "http://")
	if i := strings.IndexAny(s, "/?#"); i >= 0 {
		s = s[:i]
	}
	return strings.TrimPrefix(s, "www.")
}

// ensureLinkPreview — ambil pratinjau tautan + thumbnail (async, sekali per URL).
func (u *UI) ensureLinkPreview(url string) {
	if u.core == nil || url == "" {
		return
	}
	u.linkMu.Lock()
	if u.linkTried[url] {
		u.linkMu.Unlock()
		return
	}
	u.linkTried[url] = true
	u.linkMu.Unlock()
	go func() {
		dto := u.core.GetLinkPreview(url)
		if dto == nil {
			return
		}
		u.linkMu.Lock()
		u.linkPrev[url] = dto
		u.linkMu.Unlock()
		if dto.Image != "" {
			if img := decodeImage(decodeDataURI(u.core.FetchRemoteMedia(dto.Image))); img != nil {
				op := paint.NewImageOp(img)
				u.linkMu.Lock()
				u.linkImg[url] = op
				u.linkMu.Unlock()
			}
		}
	}()
}

// linkCardWidget — widget kartu pratinjau tautan bila sudah termuat; nil bila belum.
func (u *UI) linkCardWidget(url string) layout.Widget {
	u.linkMu.Lock()
	dto := u.linkPrev[url]
	img, hasImg := u.linkImg[url]
	u.linkMu.Unlock()
	if dto == nil {
		return nil
	}
	return func(gtx layout.Context) layout.Dimensions {
		rr := gtx.Dp(8)
		macro := op.Record(gtx.Ops)
		dims := func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			children := []layout.FlexChild{}
			if hasImg {
				children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					w := gtx.Constraints.Max.X
					h := gtx.Dp(120)
					box := image.Pt(w, h)
					cl := clip.RRect{Rect: image.Rectangle{Max: box}, NW: rr, NE: rr}.Push(gtx.Ops)
					drawImageFill(gtx.Ops, img, w)
					cl.Pop()
					if dto.Video { // tautan video (YouTube/TikTok/…) → badge play di tengah
						gtx.Constraints.Min, gtx.Constraints.Max = box, box
						layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							d := gtx.Dp(44)
							sz := image.Pt(d, d)
							paint.FillShape(gtx.Ops, color.NRGBA{A: 150}, clip.Ellipse{Max: sz}.Op(gtx.Ops))
							gtx.Constraints.Min, gtx.Constraints.Max = sz, sz
							layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return icon(gtx, "play", 22, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
							})
							return layout.Dimensions{Size: sz}
						})
					}
					return layout.Dimensions{Size: box}
				}))
			}
			children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min.X = gtx.Constraints.Max.X
					col := []layout.FlexChild{}
					if dto.Title != "" {
						col = append(col, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							l := material.Label(u.th, 13.5, dto.Title)
							l.Color, l.Font.Weight, l.MaxLines = u.t.Text, font.Medium, 2
							return l.Layout(gtx)
						}))
					}
					if dto.Desc != "" {
						col = append(col, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							l := material.Label(u.th, 12.5, dto.Desc)
							l.Color, l.MaxLines = u.t.Text2, 2
							return l.Layout(gtx)
						}))
					}
					col = append(col, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						site := dto.Site // nama situs ("Instagram"/"TikTok"); fallback host
						if site == "" {
							site = urlHost(url)
						}
						l := material.Label(u.th, 11.5, site)
						l.Color, l.MaxLines = u.t.Accent, 1
						return l.Layout(gtx)
					}))
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx, col...)
				})
			}))
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
		}(gtx)
		call := macro.Stop()
		bg := color.NRGBA{R: 0, G: 0, B: 0, A: 36}
		paint.FillShape(gtx.Ops, bg, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: rr, NE: rr, SE: rr, SW: rr}.Op(gtx.Ops))
		paint.FillShape(gtx.Ops, u.t.Accent, clip.Rect{Max: image.Pt(gtx.Dp(3), dims.Size.Y)}.Op())
		call.Add(gtx.Ops)
		return dims
	}
}

// isGroupJIDStr — true bila JID grup (@g.us).
func isGroupJIDStr(jid string) bool { return strings.HasSuffix(jid, "@g.us") }

// jidUser — bagian sebelum "@" dari JID (nomor / id) utk fallback nama anggota.
func jidUser(jid string) string {
	if i := strings.IndexByte(jid, '@'); i >= 0 {
		return jid[:i]
	}
	return jid
}

// typingOf — label mengetik chat (core.TypingLabel) atau override demo.
func (u *UI) typingOf(jid string) string {
	if u.demoTypingJID != "" && jid == u.demoTypingJID {
		return "mengetik…"
	}
	if u.core != nil {
		return u.core.TypingLabel(jid)
	}
	return ""
}

// typingWhoOf — (nama, jid) pengirim grup yg mengetik (utk avatar di bubble). DM
// / demo → ("",""): bubble tanpa avatar.
func (u *UI) typingWhoOf(jid string) (string, string) {
	if u.demoTypingWhoJID != "" {
		return u.demoTypingWho, u.demoTypingWhoJID
	}
	if u.core == nil || u.demoTypingJID != "" {
		return "", ""
	}
	return u.core.TypingWho(jid)
}

// SetRenameDemo — render-tool: pilih DM + buka modal edit nama kontak.
func (u *UI) SetRenameDemo() {
	for i := range u.chats {
		if !u.chats[i].Group {
			u.selected, u.selName, u.selGroup = u.chats[i].ID, u.chats[i].Name, false
			break
		}
	}
	u.renameEd.SetText(u.selName)
	u.overlay = "renamecontact"
}

// StatusVideo — sesi pemutaran video status inline (impl di cmd/whatslite-gio →
// internal/video: frame ffmpeg + audio libmpv). gioui tetap bebas-cgo via interface.
type StatusVideo interface {
	Frame(elapsed time.Duration) (img image.Image, ended bool) // frame utk waktu-main
	Duration() time.Duration
	SetPause(bool)
	Close()
}

// closeStatusVid — tutup sesi video status aktif (ganti item / tutup viewer).
func (u *UI) closeStatusVid() {
	if u.stVid != nil {
		u.stVid.Close()
		u.stVid = nil
	}
	u.stVidID = ""
}

// CloseStatusCompose — tutup composer status (dipanggil dari goroutine pick media).
func (u *UI) CloseStatusCompose() {
	if u.overlay == "statuscompose" {
		u.overlay = "status"
	}
	u.scEd.SetText("")
}

// SetStatusViewDemo — render-tool: buka viewer status (progress bar + item ke-2).
func (u *UI) SetStatusViewDemo() {
	u.statusGroupsCache = []app.StatusGroupDTO{{
		Jid: "x@s.whatsapp.net", Name: "Andi Pratama", Time: "2 menit lalu", Count: 4,
		Items: []app.StatusItemDTO{
			{ID: "s1", Type: "text", Text: "Status pertama 👋", Time: "08.00", Ts: 1},
			{ID: "s2", Type: "text", Text: "Lagi di pantai hari ini, cuaca cerah banget!", Time: "08.01", Ts: 2},
			{ID: "s3", Type: "text", Text: "Ketiga", Time: "08.02", Ts: 3},
			{ID: "s4", Type: "text", Text: "Keempat", Time: "08.03", Ts: 4},
		},
	}}
	u.statusViewIdx, u.statusItemIdx = 0, 1 // di item ke-2 → 2 bar penuh, 2 redup
	u.overlay = "statusview"
}

// SetContactCtxDemo — render-tool: buka menu konteks kontak (klik-kanan) demo.
func (u *UI) SetContactCtxDemo() {
	u.view = "contacts"
	u.cctContact = app.ContactRowDTO{JID: "628000@s.whatsapp.net", Name: "Alice", Phone: "+62 812 0000 1111"}
	u.overlay = "contactctx"
}

// SetTypingDemo — render-tool: pilih chat + paksa indikator mengetik (uji headless).
func (u *UI) SetTypingDemo(jid string) {
	if jid == "" && len(u.chats) > 0 {
		jid = u.chats[0].ID
	}
	u.selected, u.demoTypingJID = jid, jid
	if u.core != nil {
		u.messages = u.core.GetMessages(jid)
	}
}

// SetTypingDemoGroup — render-tool: indikator mengetik grup + avatar pengetik.
func (u *UI) SetTypingDemoGroup() {
	jid := ""
	for i := range u.chats {
		if u.chats[i].Group {
			jid = u.chats[i].ID
			u.selGroup = true
			break
		}
	}
	if jid == "" && len(u.chats) > 0 {
		jid = u.chats[0].ID
	}
	u.selected, u.demoTypingJID = jid, jid
	u.demoTypingWho, u.demoTypingWhoJID = "Andi", "111@s.whatsapp.net"
	if u.core != nil {
		u.messages = u.core.GetMessages(jid)
	}
}

// isChatMuted — status bisu chat dari daftar chat termuat (default false).
func (u *UI) isChatMuted(jid string) bool {
	for i := range u.chats {
		if u.chats[i].ID == jid {
			return u.chats[i].Muted
		}
	}
	return false
}

// communityRows membangun pane Komunitas dari komunitas nyata (core.GetCommunities).
// nil = demo. TTL-cache via chCache? pakai gate sendiri (jarang berubah).
func (u *UI) communityRows() []comItem {
	if u.core == nil {
		return u.comCache // demo: di-inject via SetCommunityDemo
	}
	if u.comCache != nil && time.Since(u.comAt) < 2*time.Second {
		return u.comCache
	}
	cs := u.core.GetCommunities()
	out := make([]comItem, 0, len(cs))
	for _, c := range cs {
		sub := itoa(len(c.Groups)) + " grup"
		names := make([]string, 0, 3)
		groups := make([]comSub, 0, len(c.Groups))
		for i, g := range c.Groups {
			if i < 3 {
				names = append(names, g.Name)
			}
			groups = append(groups, comSub{jid: g.JID, name: g.Name, isDefault: g.IsDefault})
		}
		if len(names) > 0 {
			sub += " · " + strings.Join(names, ", ")
		}
		out = append(out, comItem{jid: c.JID, name: c.Name, sub: sub, groups: groups})
	}
	u.comCache, u.comAt = out, time.Now()
	return out
}

// comCtl membangun ComCtl dari komunitas nyata + state buka/detail. Tangani klik
// di handleCommunities (DI LUAR layout, sebelum ini dipanggil).
func (u *UI) comCtl(rows []comItem) *ComCtl {
	ctl := &ComCtl{Items: rows, NewBtn: &u.comNewBtn, Back: &u.comBack}
	if len(u.comRowClicks) < len(rows) {
		u.comRowClicks = make([]widget.Clickable, len(rows))
	}
	ctl.RowClicks = u.comRowClicks[:len(rows)]
	if u.comOpen != "" { // cari komunitas terbuka → detail
		for i := range rows {
			if rows[i].jid == u.comOpen {
				ctl.Open = &rows[i]
				break
			}
		}
		if ctl.Open != nil {
			if len(u.comSubClicks) < len(ctl.Open.groups) {
				u.comSubClicks = make([]widget.Clickable, len(ctl.Open.groups))
			}
			ctl.SubClicks = u.comSubClicks[:len(ctl.Open.groups)]
		} else {
			u.comOpen = "" // komunitas hilang → kembali ke daftar
		}
	}
	return ctl
}

// handleCommunities — klik baris komunitas (buka detail), sub-grup (buka chat),
// kembali. Dipanggil DI LUAR layout pane (ala handleChatClicks).
func (u *UI) handleCommunities(gtx layout.Context, rows []comItem) {
	for u.comBack.Clicked(gtx) {
		u.comOpen = ""
	}
	for i := range rows {
		if i >= len(u.comRowClicks) {
			break
		}
		if u.comRowClicks[i].Clicked(gtx) {
			u.comOpen = rows[i].jid
		}
	}
	if u.comOpen == "" {
		return
	}
	var open *comItem
	for i := range rows {
		if rows[i].jid == u.comOpen {
			open = &rows[i]
			break
		}
	}
	if open == nil {
		return
	}
	for i := range open.groups {
		if i >= len(u.comSubClicks) {
			break
		}
		if u.comSubClicks[i].Clicked(gtx) { // buka chat sub-grup
			g := open.groups[i]
			u.selected, u.selName, u.selGroup = g.jid, g.name, true
			u.view = "chats"
			u.comOpen = ""
			if u.core != nil {
				u.core.OpenChat(g.jid)
				u.messages = u.core.GetMessages(g.jid)
			}
		}
	}
}

// fmtSubs — "12 pengikut" / "12,3 rb pengikut" / "1,2 jt pengikut".
func fmtSubs(n int) string {
	switch {
	case n >= 1000000:
		return itoa(n/1000000) + " jt pengikut"
	case n >= 1000:
		return itoa(n/1000) + " rb pengikut"
	default:
		return itoa(n) + " pengikut"
	}
}

// statusRows membangun baris pane Status (TERKINI) dari grup status nyata
// (core.GetStatuses), mengecualikan status sendiri (itu baris "My status"). nil =
// mode demo.
func (u *UI) statusRows() []stpItem {
	if u.core == nil {
		return nil
	}
	if u.srCache != nil && time.Since(u.srAt) < time.Second {
		return u.srCache
	}
	gs := u.core.GetStatuses()
	u.statusGroupsCache = u.statusGroupsCache[:0]
	out := make([]stpItem, 0, len(gs))
	for _, g := range gs {
		if g.Mine {
			continue // status sendiri tampil di baris My-status, bukan daftar
		}
		u.statusGroupsCache = append(u.statusGroupsCache, g)
		out = append(out, stpItem{name: g.Name, time: g.Time, seen: g.Seen, jid: g.Jid, count: g.Count, seenCount: g.SeenCount})
	}
	if len(u.statusClicks) < len(out) {
		u.statusClicks = make([]widget.Clickable, len(out))
	}
	u.srCache, u.srAt = out, time.Now()
	return out
}

// statusViewLayer — penampil status layar penuh: bar atas (nama+waktu+tutup) +
// item pertama (teks besar di tengah / gambar dari thumb data-URI).
func (u *UI) statusViewLayer(gtx layout.Context) {
	if u.statusViewIdx < 0 || u.statusViewIdx >= len(u.statusGroupsCache) {
		u.overlay = ""
		return
	}
	g := u.statusGroupsCache[u.statusViewIdx]
	n := len(g.Items)
	if n == 0 {
		u.overlay = ""
		return
	}
	if u.statusItemIdx >= n {
		u.statusItemIdx = n - 1
	}
	if u.statusItemIdx < 0 {
		u.statusItemIdx = 0
	}
	// next/prev item; di akhir → tutup. Reset timer auto-advance tiap pindah.
	goItem := func(delta int) {
		ni := u.statusItemIdx + delta
		if ni < 0 {
			u.statusItemIdx = 0
			return
		}
		if ni >= n {
			u.overlay = "" // habis → tutup
			u.closeStatusVid()
			return
		}
		u.statusItemIdx = ni
		u.stItemStart = gtx.Now
		if u.core != nil { // tandai item dilihat (cincin abu bertambah)
			u.core.MarkStatusSeen(g.Jid, g.Items[ni].Ts)
			u.srAt = time.Time{}
		}
	}
	for u.statusClose.Clicked(gtx) {
		u.overlay = ""
		u.closeStatusVid()
	}
	for u.stPause.Clicked(gtx) { // jeda / lanjut auto-advance
		u.stPaused = !u.stPaused
		u.stItemStart = gtx.Now // reset hitung saat lanjut
	}
	for u.stNextZone.Clicked(gtx) {
		goItem(1)
	}
	for u.stPrevZone.Clicked(gtx) {
		goItem(-1)
	}
	item := g.Items[u.statusItemIdx]
	isVideo := item.Type == "video"
	// video → buka sesi inline (frame+audio) sekali per item; ganti item tutup lama.
	if isVideo && u.stVidID != item.ID {
		u.closeStatusVid()
		if u.OnStatusVideo != nil {
			u.stVid = u.OnStatusVideo(item.ID)
			u.stVidID = item.ID
		}
	} else if !isVideo && u.stVid != nil {
		u.closeStatusVid()
	}
	for u.stFwd.Clicked(gtx) { // forward isi status → pilih chat
		u.fwdMsgID, u.fwdSrc = item.ID, "status@broadcast"
		u.stPaused = true
		u.overlay = "forward"
		u.closeStatusVid()
	}
	if gtx.Focused(&u.stReplyEd) { // sedang mengetik balasan → jeda (jangan loncat)
		u.stPaused = true
	}
	if u.stVid != nil {
		u.stVid.SetPause(u.stPaused) // sinkron audio dgn pause viewer
	}
	// durasi item: video = durasi klip; lainnya = 5 dtk. Auto-advance saat penuh.
	itemDur := 5 * time.Second
	if isVideo && u.stVid != nil {
		if d := u.stVid.Duration(); d > 0 {
			itemDur = d
		}
	}
	dur := float32(0)
	if !u.stPaused {
		el := gtx.Now.Sub(u.stItemStart)
		dur = float32(el) / float32(itemDur)
		if dur >= 1 {
			goItem(1)
			dur = 0
		} else {
			gtx.Execute(op.InvalidateCmd{}) // animasikan bar + frame video
		}
	}
	// reaksi emoji cepat (ala IG story) → ReactStatus.
	for i := range u.stEmoji {
		for u.stEmoji[i].Clicked(gtx) {
			if u.core != nil {
				u.core.ReactStatus(g.Jid, item.ID, statusEmojis[i])
				u.stReplied = statusEmojis[i] // umpan balik singkat
			}
			u.stPaused = true
		}
	}
	for u.stEmojiMore.Clicked(gtx) { // "+" → picker emoji lengkap utk reaksi status
		u.stPaused = true
		if u.stVid != nil {
			u.stVid.SetPause(true) // jeda audio video selama picker terbuka
		}
		u.overlay = "statusemoji"
	}
	// balas teks → ReplyStatus (DM ke poster, mengutip status).
	sendReply := func() {
		t := strings.TrimSpace(u.stReplyEd.Text())
		if t != "" && u.core != nil {
			u.core.ReplyStatus(g.Jid, item.ID, item.Text, t)
			u.stReplyEd.SetText("")
			u.stReplied = "✓"
		}
	}
	for u.stReplySend.Clicked(gtx) {
		sendReply()
	}
	for {
		ev, ok := u.stReplyEd.Update(gtx)
		if !ok {
			break
		}
		if _, ok := ev.(widget.SubmitEvent); ok {
			sendReply()
		}
	}
	paint.FillShape(gtx.Ops, color.NRGBA{R: 0x0b, G: 0x14, B: 0x1a, A: 0xff}, clip.Rect{Max: gtx.Constraints.Max}.Op()) // latar solid (tanpa app tembus)
	gtx.Constraints.Min = gtx.Constraints.Max                                                                           // isi penuh layar → konten ter-center vertikal
	layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		// progress bar tersegmen (1 per item; <=current penuh, sisanya redup).
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(10), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min.X = gtx.Constraints.Max.X
				gap := gtx.Dp(3)
				h := gtx.Dp(3)
				total := gtx.Constraints.Max.X
				bw := (total - gap*(n-1)) / n
				if bw < 1 {
					bw = 1
				}
				dim := color.NRGBA{R: 255, G: 255, B: 255, A: 0x55}
				full := color.NRGBA{R: 255, G: 255, B: 255, A: 0xff}
				rr := func(x0, x1 int, c color.NRGBA) {
					paint.FillShape(gtx.Ops, c, clip.RRect{Rect: image.Rectangle{Min: image.Pt(x0, 0), Max: image.Pt(x1, h)}, NW: h / 2, NE: h / 2, SE: h / 2, SW: h / 2}.Op(gtx.Ops))
				}
				for i := 0; i < n; i++ {
					x := i * (bw + gap)
					switch {
					case i < u.statusItemIdx: // sudah lewat → penuh
						rr(x, x+bw, full)
					case i == u.statusItemIdx: // aktif → track redup + isi sesuai progress (animasi)
						rr(x, x+bw, dim)
						fillW := int(float32(bw) * dur)
						if fillW > 0 {
							rr(x, x+fillW, full)
						} else {
							rr(x, x+bw, full) // video/jeda → tampil penuh
						}
					default: // belum → redup
						rr(x, x+bw, dim)
					}
				}
				return layout.Dimensions{Size: image.Pt(total, h)}
			})
		}),
		// bar atas: avatar + nama + waktu + tutup
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(16), Bottom: unit.Dp(8), Left: unit.Dp(16), Right: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions { return u.avatar(gtx, g.Name, g.Jid, 38) }),
					layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								l := material.Label(u.th, 15, g.Name)
								l.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
								l.Font.Weight = font.Medium
								return l.Layout(gtx)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								l := material.Label(u.th, 12, item.Time)
								l.Color = color.NRGBA{R: 0xcc, G: 0xcc, B: 0xcc, A: 0xff}
								return l.Layout(gtx)
							}),
						)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return u.stFwd.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return layout.UniformInset(unit.Dp(6)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return icon(gtx, "forward", 20, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
							})
						})
					}),
					layout.Rigid(layout.Spacer{Width: unit.Dp(4)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return u.stPause.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return layout.UniformInset(unit.Dp(6)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								ic := "pause"
								if u.stPaused {
									ic = "play"
								}
								return icon(gtx, ic, 20, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
							})
						})
					}),
					layout.Rigid(layout.Spacer{Width: unit.Dp(4)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return u.statusClose.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return layout.UniformInset(unit.Dp(6)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return icon(gtx, "close", 22, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
							})
						})
					}),
				)
			})
		}),
		// isi: media penuh (image/video/sticker) atau teks besar + zona tap navigasi.
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			fill := func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{Size: gtx.Constraints.Max} }
			return layout.Stack{}.Layout(gtx,
				layout.Expanded(func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min = gtx.Constraints.Max // isi penuh → konten ter-center
					return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						// gambar/video: video → frame inline (ffmpeg) per waktu-main;
						// image/sticker → unduh media penuh. Fallback thumb tertanam.
						var iop paint.ImageOp
						haveImg := false
						if isVideo && u.stVid != nil {
							if fr, _ := u.stVid.Frame(gtx.Now.Sub(u.stItemStart)); fr != nil {
								iop, haveImg = paint.NewImageOp(fr), true
							}
						} else if item.Type == "image" || item.Type == "sticker" {
							u.ensureMedia("status@broadcast", item.ID, item.Type)
							u.mediaMu.Lock()
							if op2, ok := u.media[item.ID]; ok {
								iop, haveImg = op2, true
							}
							u.mediaMu.Unlock()
							// belum termuat → redraw terbatas ~5s agar muncul saat unduh selesai.
							if !haveImg && gtx.Now.Sub(u.statusViewAt) < 5*time.Second {
								gtx.Execute(op.InvalidateCmd{})
							}
						}
						if !haveImg {
							if img := decodeImage(decodeDataURI(item.Thumb)); img != nil {
								iop, haveImg = paint.NewImageOp(img), true
							}
						}
						if haveImg {
							sz := iop.Size()
							maxW, maxH := gtx.Dp(420), gtx.Constraints.Max.Y-gtx.Dp(80)
							w := maxW
							h := w * sz.Y / sz.X
							if h > maxH {
								h = maxH
								w = h * sz.X / sz.Y
							}
							box := image.Pt(w, h)
							cl := clip.Rect{Max: box}.Push(gtx.Ops)
							drawImageFill(gtx.Ops, iop, w)
							cl.Pop()
							return layout.Dimensions{Size: box}
						}
						txt := item.Text
						if txt == "" {
							txt = "Status"
						}
						return layout.Inset{Left: unit.Dp(40), Right: unit.Dp(40)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							l := material.Label(u.th, 26, txt)
							l.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
							l.Alignment = text.Middle
							return l.Layout(gtx)
						})
					})
				}),
				layout.Expanded(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
						layout.Flexed(0.32, func(gtx layout.Context) layout.Dimensions { return u.stPrevZone.Layout(gtx, fill) }),
						layout.Flexed(0.68, func(gtx layout.Context) layout.Dimensions { return u.stNextZone.Layout(gtx, fill) }),
					)
				}),
			)
		}),
		// bilah bawah: reaksi emoji cepat + kotak balas (status milik sendiri tak ada).
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if g.Mine {
				return layout.Dimensions{}
			}
			white := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
			return layout.Inset{Top: unit.Dp(6), Bottom: unit.Dp(16), Left: unit.Dp(20), Right: unit.Dp(20)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min.X = gtx.Constraints.Max.X
				return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
					// baris emoji reaksi cepat.
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						gtx.Constraints.Min.X = gtx.Constraints.Max.X
						children := emojiBtns(gtx, u.th, u.stEmoji[:])
						children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return u.stEmojiMore.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return layout.UniformInset(unit.Dp(6)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									d := gtx.Dp(34)
									sz := image.Pt(d, d)
									paint.FillShape(gtx.Ops, color.NRGBA{R: 255, G: 255, B: 255, A: 0x22}, clip.Ellipse{Max: sz}.Op(gtx.Ops))
									gtx.Constraints.Min, gtx.Constraints.Max = sz, sz
									return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
										return icon(gtx, "plus", 20, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
									})
								})
							})
						}))
						return layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceEvenly}.Layout(gtx, children...)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
					// kotak balas (pill gelap + editor + tombol kirim).
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						gtx.Constraints.Min.X = gtx.Constraints.Max.X
						macro := op.Record(gtx.Ops)
						dims := layout.Inset{Top: unit.Dp(9), Bottom: unit.Dp(9), Left: unit.Dp(16), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
								layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
									e := material.Editor(u.th, &u.stReplyEd, "Balas…")
									e.Color, e.HintColor, e.TextSize = white, color.NRGBA{R: 0xbb, G: 0xbb, B: 0xbb, A: 0xff}, unit.Sp(15)
									return e.Layout(gtx)
								}),
								layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return u.stReplySend.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
										return icon(gtx, "send", 22, u.t.Accent)
									})
								}),
							)
						})
						call := macro.Stop()
						rr := dims.Size.Y / 2
						paint.FillShape(gtx.Ops, color.NRGBA{R: 0x2a, G: 0x35, B: 0x3c, A: 0xff}, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: rr, NE: rr, SE: rr, SW: rr}.Op(gtx.Ops))
						call.Add(gtx.Ops)
						return dims
					}),
				)
			})
		}),
	)
}

// statusComposeLayer — composer status teks sendiri (latar accent ala WhatsApp):
// ketik → tombol kirim → PostTextStatus. (Status media via lampiran = TODO.)
func (u *UI) statusComposeLayer(gtx layout.Context) {
	paint.FillShape(gtx.Ops, u.t.Accent, clip.Rect{Max: gtx.Constraints.Max}.Op()) // kanvas warna
	white := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	post := func() {
		t := strings.TrimSpace(u.scEd.Text())
		if t != "" && u.core != nil {
			u.core.PostTextStatus(t, int64(argbOf(u.t.Accent)), 0)
		}
		u.scEd.SetText("")
		u.overlay = "status" // kembali ke pane status
	}
	for u.scCancel.Clicked(gtx) {
		u.scEd.SetText("")
		u.overlay = "status"
	}
	for u.scMedia.Clicked(gtx) { // pilih foto/video → unggah status media
		if u.OnStatusMedia != nil {
			u.OnStatusMedia()
		}
	}
	for u.scPost.Clicked(gtx) {
		post()
	}
	for {
		ev, ok := u.scEd.Update(gtx)
		if !ok {
			break
		}
		if _, ok := ev.(widget.SubmitEvent); ok {
			post()
		}
	}
	gtx.Constraints.Min = gtx.Constraints.Max
	layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		// bar atas: batal (kiri) + kirim (kanan).
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(16), Bottom: unit.Dp(8), Left: unit.Dp(16), Right: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min.X = gtx.Constraints.Max.X
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return u.scCancel.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return layout.UniformInset(unit.Dp(6)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return icon(gtx, "close", 22, white)
							})
						})
					}),
					layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
					// tombol foto/video → unggah status media.
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return u.scMedia.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return layout.UniformInset(unit.Dp(6)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return icon(gtx, "camera", 22, white)
							})
						})
					}),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						lbl := material.Label(u.th, 15, "Status saya")
						lbl.Color, lbl.Alignment = white, text.Middle
						return lbl.Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return u.scPost.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return layout.UniformInset(unit.Dp(6)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return icon(gtx, "send", 22, white)
							})
						})
					}),
				)
			})
		}),
		// area teks besar di tengah.
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Left: unit.Dp(40), Right: unit.Dp(40)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					u.scEd.Alignment = text.Middle
					e := material.Editor(u.th, &u.scEd, "Ketik status…")
					e.Color, e.HintColor, e.TextSize = white, color.NRGBA{R: 255, G: 255, B: 255, A: 0xaa}, unit.Sp(26)
					return e.Layout(gtx)
				})
			})
		}),
	)
}

// argbOf — warna → ARGB uint32 (utk bg status teks).
func argbOf(c color.NRGBA) uint32 {
	return uint32(c.A)<<24 | uint32(c.R)<<16 | uint32(c.G)<<8 | uint32(c.B)
}

// statusEmojiLayer — picker emoji lengkap utk reaksi status (tombol "+"). Pilih →
// ReactStatus lalu kembali ke viewer.
func (u *UI) statusEmojiLayer(gtx layout.Context) {
	if u.statusViewIdx < 0 || u.statusViewIdx >= len(u.statusGroupsCache) {
		u.overlay = "statusview"
		return
	}
	g := u.statusGroupsCache[u.statusViewIdx]
	if u.statusItemIdx < len(g.Items) {
		item := g.Items[u.statusItemIdx]
		em := RpEmoji()
		for i := range u.rpClicks {
			if i >= len(em) {
				break
			}
			for u.rpClicks[i].Clicked(gtx) {
				if u.core != nil {
					u.core.ReactStatus(g.Jid, item.ID, em[i])
				}
				u.overlay = "statusview" // kembali ke viewer
			}
		}
	}
	for u.backdrop.Clicked(gtx) {
		u.overlay = "statusview"
	}
	u.backdrop.Layout(gtx, func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{Size: gtx.Constraints.Max} })
	ReactionPickerView(gtx, u.th, u.t, &RpCtl{Clicks: u.rpClicks, List: &u.rpList})
}

// statusEmojis — reaksi cepat status (ala IG story / WhatsApp).
var statusEmojis = [6]string{"👍", "❤️", "😂", "😮", "😢", "🙏"}

// emojiBtns — 6 tombol emoji reaksi status (label besar, clickable).
func emojiBtns(gtx layout.Context, th *material.Theme, clicks []widget.Clickable) []layout.FlexChild {
	out := make([]layout.FlexChild, 0, len(statusEmojis))
	for i := range statusEmojis {
		i := i
		out = append(out, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if i >= len(clicks) {
				return layout.Dimensions{}
			}
			return clicks[i].Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.UniformInset(unit.Dp(4)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					l := material.Label(th, 30, statusEmojis[i])
					return l.Layout(gtx)
				})
			})
		}))
	}
	return out
}

// decodeDataURI — "data:<mime>;base64,<...>" → byte. "" bila bukan data-URI base64.
func decodeDataURI(s string) []byte {
	i := strings.Index(s, ";base64,")
	if i < 0 || !strings.HasPrefix(s, "data:") {
		return nil
	}
	b, err := base64.StdEncoding.DecodeString(s[i+len(";base64,"):])
	if err != nil {
		return nil
	}
	return b
}

// handleStatus — klik baris status → buka viewer (overlay statusview).
func (u *UI) handleStatus(gtx layout.Context) {
	for u.stMyClick.Clicked(gtx) { // "Status saya" → composer post status sendiri
		u.scEd.SetText("")
		u.overlay = "statuscompose"
	}
	for i := range u.statusGroupsCache {
		if i >= len(u.statusClicks) {
			break
		}
		for u.statusClicks[i].Clicked(gtx) {
			u.statusViewIdx = i
			u.statusItemIdx = 0 // mulai dari item paling lama
			u.statusViewAt = gtx.Now
			u.stItemStart = gtx.Now
			u.stPaused = false
			u.overlay = "statusview"
			g := u.statusGroupsCache[i] // tandai item pertama dilihat
			if u.core != nil && len(g.Items) > 0 {
				u.core.MarkStatusSeen(g.Jid, g.Items[0].Ts)
				u.srAt = time.Time{} // invalidasi cache cincin
			}
		}
	}
}

// infoJID — sasaran efektif drawer info: kontak yg sedang di-INTIP (infoCJID) bila
// dibuka dari ikon "i" pane Kontak, selain itu chat terpilih. infoNameOf serupa.
func (u *UI) infoJID() string {
	if u.infoCJID != "" {
		return u.infoCJID
	}
	return u.selected
}
func (u *UI) infoNameOf() string {
	if u.infoCJID != "" {
		return u.infoCName
	}
	return u.selName
}

// infoData membangun data drawer info dari sasaran efektif (intip-kontak / chat
// terpilih). nil = demo. GetGroupInfo hanya dipanggil saat drawer dibuka.
func (u *UI) infoData() *InfoDrawerData {
	jid := u.infoJID()
	if u.core == nil || jid == "" {
		return nil
	}
	group := u.selGroup && u.infoCJID == "" // intip-kontak selalu DM (bukan grup)
	sub := u.subtitle
	if u.infoCJID != "" { // intip-kontak: subtitle = nomor/about (bukan presence chat)
		sub = u.core.GetContactAbout(jid)
	}
	d := &InfoDrawerData{Name: u.infoNameOf(), Group: group, Sub: sub}
	d.Mute = &u.infoMuteC // bisu (DM + grup)
	d.Media = &u.infoMediaC
	d.Enc = &u.infoEncC
	d.Timer = &u.infoTimerC
	d.TimerLabel = dispLabel(u.dispTimer[jid])
	d.Muted = u.isChatMuted(jid)
	if group {
		d.Leave = &u.infoLeaveC
		d.Invite = &u.infoInviteC
		d.Edit = &u.infoEditC
		d.Add = &u.infoAddC
		if gi := u.core.GetGroupInfo(jid); gi != nil {
			d.Sub = itoa(len(gi.Participants)) + " anggota"
			d.Desc = gi.Topic
			u.curGroupDesc = gi.Topic
			u.curGroupAmAdmin = gi.AmAdmin
			d.AmAdmin = gi.AmAdmin
			d.Announce, d.Locked, d.Approval = gi.Announce, gi.Locked, gi.JoinApproval
			d.AnnounceC, d.LockedC, d.ApprovalC = &u.infoAnnounceC, &u.infoLockedC, &u.infoApprovalC
			d.Members = make([]InfoMember, 0, len(gi.Participants))
			u.infoMemberJIDs = u.infoMemberJIDs[:0]
			u.infoMemberNames = u.infoMemberNames[:0]
			u.infoMemberAdmin = u.infoMemberAdmin[:0]
			for _, p := range gi.Participants {
				nm := p.Name
				if nm == "" {
					nm = jidUser(p.JID)
				}
				d.Members = append(d.Members, InfoMember{Name: nm, Admin: p.IsAdmin, JID: p.JID})
				u.infoMemberJIDs = append(u.infoMemberJIDs, p.JID)
				u.infoMemberNames = append(u.infoMemberNames, nm)
				u.infoMemberAdmin = append(u.infoMemberAdmin, p.IsAdmin)
			}
			if len(u.infoMemberClicks) < len(d.Members) {
				u.infoMemberClicks = make([]widget.Clickable, len(d.Members))
			}
			d.MemberClicks = u.infoMemberClicks[:len(d.Members)]
		}
	} else {
		d.Block = &u.infoBlockC
		d.Rename = &u.infoRenameC
		d.Blocked = u.core.IsBlocked(jid)
		if d.Desc == "" { // "Tentang" kontak (about) bila ada
			d.Desc = sub
		}
		u.ensureCommonGroups(jid) // grup bersama (async)
		u.commonMu.Lock()
		d.CommonGroups = u.commonGroups[jid]
		u.commonMu.Unlock()
		if len(u.commonClicks) < len(d.CommonGroups) {
			u.commonClicks = make([]widget.Clickable, len(d.CommonGroups))
		}
		d.CommonClicks = u.commonClicks[:len(d.CommonGroups)]
	}
	return d
}

// handleInfo — aksi tombol info-drawer: blokir kontak / keluar grup, lalu tutup.
func (u *UI) handleInfo(gtx layout.Context) {
	jid := u.infoJID() // sasaran: kontak yg diintip ("i") atau chat terpilih
	for u.infoBlockC.Clicked(gtx) {
		if u.core != nil {
			u.core.Block(jid, !u.core.IsBlocked(jid)) // toggle blokir
		}
		u.overlay, u.infoCJID = "", ""
	}
	for u.infoRenameC.Clicked(gtx) { // edit nama kontak → modal (sasaran = jid efektif)
		u.renameEd.SetText(u.infoNameOf())
		u.renameTarget = jid
		u.overlay = "renamecontact"
	}
	for u.infoLeaveC.Clicked(gtx) {
		if u.core != nil {
			u.core.LeaveGroup(u.selected)
		}
		u.overlay, u.selected, u.infoCJID = "", "", "" // keluar → tutup drawer + deselect chat
	}
	for u.infoEditC.Clicked(gtx) { // edit info grup → modal nama+deskripsi
		u.gedName.SetText(u.selName)
		u.gedDesc.SetText(u.curGroupDesc)
		u.overlay = "groupedit"
	}
	for u.infoMuteC.Clicked(gtx) { // bisukan / aktifkan notifikasi
		if u.core != nil {
			u.core.Mute(jid, !u.isChatMuted(jid))
		}
	}
	for u.infoMediaC.Clicked(gtx) { // galeri media chat
		u.overlay = "media"
	}
	for u.infoEncC.Clicked(gtx) { // info enkripsi end-to-end
		u.overlay = "encryption"
	}
	for u.infoTimerC.Clicked(gtx) { // pesan sementara → picker
		u.overlay = "disappearing"
	}
	for u.infoAnnounceC.Clicked(gtx) { // toggle hanya-admin-boleh-kirim
		if u.core != nil {
			if gi := u.core.GetGroupInfo(u.selected); gi != nil {
				u.core.SetGroupAnnounce(u.selected, !gi.Announce)
			}
		}
	}
	for u.infoLockedC.Clicked(gtx) { // toggle hanya-admin-boleh-ubah-info
		if u.core != nil {
			if gi := u.core.GetGroupInfo(u.selected); gi != nil {
				u.core.SetGroupLocked(u.selected, !gi.Locked)
			}
		}
	}
	for u.infoApprovalC.Clicked(gtx) { // toggle setujui-anggota-baru
		if u.core != nil {
			if gi := u.core.GetGroupInfo(u.selected); gi != nil {
				u.core.SetGroupJoinApproval(u.selected, !gi.JoinApproval)
			}
		}
	}
	for u.infoAddC.Clicked(gtx) { // "Tambah anggota" → pemilih kontak (mode addmember)
		u.gcMode, u.gcGroupJID = "addmember", u.selected
		u.gcSel = map[string]bool{}
		u.gcNameEd.SetText("")
		u.overlay = "groupcreate"
	}
	for i := range u.infoMemberClicks { // ketuk anggota grup → menu konteks anggota
		if i >= len(u.infoMemberJIDs) {
			break
		}
		if u.infoMemberClicks[i].Clicked(gtx) {
			jid := u.infoMemberJIDs[i]
			if u.selfJID != "" && jidUser(jid) == jidUser(u.selfJID) {
				continue // diri sendiri → tak ada menu
			}
			u.mctJID = jid
			if i < len(u.infoMemberNames) {
				u.mctName = u.infoMemberNames[i]
			}
			if i < len(u.infoMemberAdmin) {
				u.mctAdmin = u.infoMemberAdmin[i]
			}
			u.overlay = "memberctx"
		}
	}
	u.handleMemberCtx(gtx)
	for i := range u.commonClicks { // grup bersama → buka grup itu
		dmJID := u.infoJID()
		u.commonMu.Lock()
		gs := u.commonGroups[dmJID]
		u.commonMu.Unlock()
		if i >= len(gs) {
			break
		}
		if u.commonClicks[i].Clicked(gtx) {
			g := gs[i]
			u.selected, u.selName, u.selGroup = g.JID, g.Name, true
			u.overlay, u.infoCJID = "", ""
			if u.core != nil {
				u.core.OpenChat(g.JID)
				u.messages = u.core.GetMessages(g.JID)
			}
		}
	}
	for u.infoInviteC.Clicked(gtx) { // link undangan → ambil async, tampil modal
		u.inviteLink = ""
		u.overlay = "invitelink"
		if u.core != nil {
			jid := u.selected
			go func() {
				link := u.core.GroupInviteLink(jid, false)
				u.inviteLink = link
			}()
		}
	}
}

// callRows membangun baris pane Panggilan dari log nyata (core.GetCalls). nil =
// mode demo (render standalone). Nama sudah di-resolve ulang di GetCalls.
func (u *UI) callRows() []spCall {
	if u.core == nil {
		return nil
	}
	if u.crCache != nil && time.Since(u.crAt) < time.Second {
		return u.crCache
	}
	cs := u.core.GetCalls()
	out := make([]spCall, 0, len(cs))
	for _, c := range cs {
		out = append(out, spCall{
			name:   c.Name,
			time:   time.Unix(c.TS, 0).Format("15.04"),
			video:  c.Video,
			missed: c.Status == "missed",
		})
	}
	u.crCache, u.crAt = out, time.Now()
	return out
}

// maybeLoadOlder — bila daftar pesan tergulir mendekati ATAS, minta 50 pesan lebih
// lama dari engine (history on-demand WhatsApp; prinsip lazy-history Telegram).
// Throttle per-chat 3s; respons tiba via OnHistorySync ON_DEMAND → GetMessages.
func (u *UI) maybeLoadOlder() {
	if u.core == nil || u.selected == "" || len(u.messages) < 3 {
		return
	}
	if u.msgList.Position.First > 3 { // belum di dekat atas
		return
	}
	u.requestOlder(u.selected)
}

// prefetchHistory — saat buka chat dgn history lokal TIPIS (cuma bootstrap recent),
// tarik satu halaman lama SEGERA → user lihat backfill TERURUT, bukan "cuma pesan
// baru". Constraint WhatsApp: server tak simpan history → minta ke HP; tiru UX
// Telegram = local store permanen + lazy ordered paging.
func (u *UI) prefetchHistory(jid string) {
	if u.core == nil || jid == "" || len(u.messages) >= 25 {
		return
	}
	u.requestOlder(jid)
}

// requestOlder — minta history lama 1× per chat (throttle 3s; respons via
// OnHistorySync ON_DEMAND → GetMessages, terurut sebelum pesan tertua).
func (u *UI) requestOlder(jid string) {
	if u.core == nil || !u.core.HasMoreHistory(jid) { // HP sudah habis → jangan minta
		return
	}
	if u.olderReqChat == jid && time.Since(u.olderReqAt) < 3*time.Second {
		return
	}
	u.olderReqChat, u.olderReqAt = jid, time.Now()
	go u.core.LoadOlderHistory(jid)
}

// scrollFab — tombol bulat gulir-ke-bawah (.scroll-fab: 42, in-bg, text2, kanan-
// bawah 18/16). Tampil hanya saat daftar tergulir naik (belum di dasar viewport).
// Diletakkan absolut di pojok kanan-bawah area daftar. (Badge "pesan baru" Svelte
// di-skip: butuh pelacakan newCount andal; FAB polos = perilaku WA paling umum.)
// fabHidden — true bila tombol gulir-ke-bawah harus disembunyikan: tak ada pesan,
// daftar belum terukur (Count==0, hindari kedip saat buka), atau pesan terakhir
// sudah tampak penuh (di dasar). Pure → bisa diuji.
func fabHidden(pos layout.Position, n int) bool {
	if n == 0 || pos.Count == 0 {
		return true
	}
	return pos.First+pos.Count >= n && pos.OffsetLast <= 0
}

func (u *UI) scrollFab(gtx layout.Context) layout.Dimensions {
	full := gtx.Constraints.Max
	for u.fabClick.Clicked(gtx) {
		u.msgList.ScrollTo(len(u.messages)) // lompat ke pesan terbaru
	}
	if fabHidden(u.msgList.Position, len(u.messages)) {
		return layout.Dimensions{Size: full}
	}
	d := gtx.Dp(42)
	x := full.X - gtx.Dp(18) - d
	y := full.Y - gtx.Dp(16) - d

	off := op.Offset(image.Pt(x, y)).Push(gtx.Ops)
	cgtx := gtx
	cgtx.Constraints.Min, cgtx.Constraints.Max = image.Pt(d, d), image.Pt(d, d)
	u.fabClick.Layout(cgtx, func(gtx layout.Context) layout.Dimensions {
		paint.FillShape(gtx.Ops, u.t.InBg, clip.Ellipse{Max: image.Pt(d, d)}.Op(gtx.Ops))
		return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return icon(gtx, "chevrondown", 24, u.t.Text2)
		})
	})
	off.Pop()
	return layout.Dimensions{Size: full}
}

// ---- percakapan (header + bubble + composer) ----
// syncDraft — saat chat aktif berganti: simpan teks composer chat lama ke drafts,
// lalu muat draft chat baru ke editor (ala WhatsApp draft per-chat). Batalkan
// mode edit/balas yg melekat ke chat lama.
func (u *UI) syncDraft() {
	if u.selected == u.draftChat {
		return
	}
	if u.draftChat != "" { // simpan draft chat sebelumnya
		t := u.editor.Text()
		if strings.TrimSpace(t) == "" {
			delete(u.drafts, u.draftChat)
		} else {
			u.drafts[u.draftChat] = t
		}
	}
	u.clearReply()
	u.clearEdit()
	u.editor.SetText(u.drafts[u.selected]) // "" bila tak ada draft
	u.draftChat = u.selected
}

// mediaPreviewLayer — pratinjau media sebelum kirim: thumbnail + caption + toggle
// sekali-lihat (image/video) + Batal/Kirim. Kirim → SendMedia(caption, viewOnce).
func (u *UI) mediaPreviewLayer(gtx layout.Context) layout.Dimensions {
	paint.FillShape(gtx.Ops, color.NRGBA{A: 150}, clip.Rect{Max: gtx.Constraints.Max}.Op())
	u.pendMu.Lock()
	kind, uri, img, hasImg, vo := u.pendKind, u.pendURI, u.pendImg, u.pendImgHas, u.pendVO
	u.pendMu.Unlock()

	for u.pendCancel.Clicked(gtx) {
		u.clearPending()
		u.overlay = ""
	}
	for u.pendVOClick.Clicked(gtx) {
		u.pendMu.Lock()
		u.pendVO = !u.pendVO
		u.pendMu.Unlock()
	}
	for u.pendRotate.Clicked(gtx) { // putar gambar 90°
		if uri != "" {
			if nu, nop, ok := rotateDataURI90(uri); ok {
				u.pendMu.Lock()
				u.pendURI, u.pendImg, u.pendImgHas = nu, nop, true
				u.pendMu.Unlock()
				uri, img, hasImg = nu, nop, true
				u.cropActive = false // dimensi berubah → buang seleksi
			}
		}
	}
	for u.pendSend.Clicked(gtx) {
		if u.core != nil && u.selected != "" {
			u.core.SendMedia(u.selected, kind, strings.TrimSpace(u.capEd.Text()), "", uri, vo, 0)
			u.messages = u.core.GetMessages(u.selected)
			u.msgList.ScrollTo(len(u.messages))
		}
		u.clearPending()
		u.overlay = ""
	}
	canVO := kind == "image" || kind == "video"

	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		w := gtx.Dp(360)
		gtx.Constraints.Min.X, gtx.Constraints.Max.X = w, w
		macro := op.Record(gtx.Ops)
		dims := layout.UniformInset(unit.Dp(16)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			children := []layout.FlexChild{
				// thumbnail (image: contain-fit + seret-potong) atau kotak ikon+jenis.
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					bw := gtx.Constraints.Max.X
					bh := gtx.Dp(200)
					box := image.Pt(bw, bh)
					r := gtx.Dp(8)
					cl := clip.RRect{Rect: image.Rectangle{Max: box}, NW: r, NE: r, SE: r, SW: r}.Push(gtx.Ops)
					paint.FillShape(gtx.Ops, u.t.Bg2, clip.Rect{Max: box}.Op())
					if !hasImg {
						gtx.Constraints.Min, gtx.Constraints.Max = box, box
						layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							ic := "docfile"
							if kind == "video" {
								ic = "play"
							}
							return icon(gtx, ic, 48, u.t.Text2)
						})
						cl.Pop()
						return layout.Dimensions{Size: box}
					}
					// contain-fit: hitung rect gambar dlm box (utk pemetaan potong tepat).
					is := img.Size()
					sc := float32(bw) / float32(is.X)
					if sy := float32(bh) / float32(is.Y); sy < sc {
						sc = sy
					}
					dispW, dispH := int(float32(is.X)*sc), int(float32(is.Y)*sc)
					ox, oy := (bw-dispW)/2, (bh-dispH)/2
					io := op.Offset(image.Pt(ox, oy)).Push(gtx.Ops)
					af := op.Affine(f32.Affine2D{}.Scale(f32.Pt(0, 0), f32.Pt(sc, sc))).Push(gtx.Ops)
					img.Add(gtx.Ops)
					paint.PaintOp{}.Add(gtx.Ops)
					af.Pop()
					io.Pop()

					imgRect := image.Rect(ox, oy, ox+dispW, oy+dispH)
					clamp := func(p image.Point) image.Point {
						if p.X < imgRect.Min.X {
							p.X = imgRect.Min.X
						}
						if p.X > imgRect.Max.X {
							p.X = imgRect.Max.X
						}
						if p.Y < imgRect.Min.Y {
							p.Y = imgRect.Min.Y
						}
						if p.Y > imgRect.Max.Y {
							p.Y = imgRect.Max.Y
						}
						return p
					}
					tag := &u.cropTagV
					for {
						ev, ok := gtx.Event(pointer.Filter{Target: tag, Kinds: pointer.Press | pointer.Drag | pointer.Release})
						if !ok {
							break
						}
						pe, ok := ev.(pointer.Event)
						if !ok {
							continue
						}
						p := clamp(image.Pt(int(pe.Position.X), int(pe.Position.Y)))
						switch pe.Kind {
						case pointer.Press:
							u.cropA, u.cropB, u.cropActive = p, p, true
						case pointer.Drag:
							u.cropB = p
						}
					}
					area := clip.Rect{Max: box}.Push(gtx.Ops)
					event.Op(gtx.Ops, tag)
					area.Pop()

					// seleksi: redupkan luar + bingkai accent.
					if u.cropActive {
						sel := image.Rectangle{Min: u.cropA, Max: u.cropB}.Canon()
						dark := color.NRGBA{A: 120}
						paint.FillShape(gtx.Ops, dark, clip.Rect{Min: image.Pt(0, 0), Max: image.Pt(bw, sel.Min.Y)}.Op())
						paint.FillShape(gtx.Ops, dark, clip.Rect{Min: image.Pt(0, sel.Max.Y), Max: box}.Op())
						paint.FillShape(gtx.Ops, dark, clip.Rect{Min: image.Pt(0, sel.Min.Y), Max: image.Pt(sel.Min.X, sel.Max.Y)}.Op())
						paint.FillShape(gtx.Ops, dark, clip.Rect{Min: image.Pt(sel.Max.X, sel.Min.Y), Max: image.Pt(bw, sel.Max.Y)}.Op())
						bd := gtx.Dp(2)
						ac := u.t.Accent
						paint.FillShape(gtx.Ops, ac, clip.Rect{Min: sel.Min, Max: image.Pt(sel.Max.X, sel.Min.Y+bd)}.Op())
						paint.FillShape(gtx.Ops, ac, clip.Rect{Min: image.Pt(sel.Min.X, sel.Max.Y-bd), Max: sel.Max}.Op())
						paint.FillShape(gtx.Ops, ac, clip.Rect{Min: sel.Min, Max: image.Pt(sel.Min.X+bd, sel.Max.Y)}.Op())
						paint.FillShape(gtx.Ops, ac, clip.Rect{Min: image.Pt(sel.Max.X-bd, sel.Min.Y), Max: sel.Max}.Op())
					}
					cl.Pop()

					// terapkan potong: seleksi box → piksel gambar.
					for u.pendCrop.Clicked(gtx) {
						sel := image.Rectangle{Min: u.cropA, Max: u.cropB}.Canon()
						pr := image.Rect(
							int(float32(sel.Min.X-ox)/sc), int(float32(sel.Min.Y-oy)/sc),
							int(float32(sel.Max.X-ox)/sc), int(float32(sel.Max.Y-oy)/sc))
						if nu, nop, ok := cropDataURI(uri, pr); ok {
							u.pendMu.Lock()
							u.pendURI, u.pendImg, u.pendImgHas = nu, nop, true
							u.pendMu.Unlock()
							uri, img, hasImg = nu, nop, true
							u.cropActive = false
						}
					}

					// tombol kanan-atas: putar; + potong bila ada seleksi.
					if kind == "image" {
						d := gtx.Dp(34)
						roundBtn := func(c *widget.Clickable, x int, ic string) {
							off := op.Offset(image.Pt(x, gtx.Dp(8))).Push(gtx.Ops)
							bgtx := gtx
							bgtx.Constraints.Min, bgtx.Constraints.Max = image.Pt(d, d), image.Pt(d, d)
							c.Layout(bgtx, func(gtx layout.Context) layout.Dimensions {
								paint.FillShape(gtx.Ops, color.NRGBA{A: 150}, clip.Ellipse{Max: image.Pt(d, d)}.Op(gtx.Ops))
								return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return icon(gtx, ic, 18, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
								})
							})
							off.Pop()
						}
						roundBtn(&u.pendRotate, bw-d-gtx.Dp(8), "rotate")
						if u.cropActive {
							roundBtn(&u.pendCrop, bw-2*d-gtx.Dp(14), "check")
						}
					}
					return layout.Dimensions{Size: box}
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
				// caption.
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					macro := op.Record(gtx.Ops)
					d := layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						gtx.Constraints.Min.X = gtx.Constraints.Max.X
						e := material.Editor(u.th, &u.capEd, "Tambah keterangan…")
						e.Color, e.HintColor, e.TextSize = u.t.Text, u.t.Text2, unit.Sp(15)
						return e.Layout(gtx)
					})
					call := macro.Stop()
					rr := gtx.Dp(8)
					paint.FillShape(gtx.Ops, u.t.SearchBg, clip.RRect{Rect: image.Rectangle{Max: d.Size}, NW: rr, NE: rr, SE: rr, SW: rr}.Op(gtx.Ops))
					call.Add(gtx.Ops)
					return d
				}),
			}
			if canVO { // toggle sekali-lihat.
				children = append(children,
					layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return u.pendVOClick.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									col := u.t.Text2
									if vo {
										col = u.t.Accent
									}
									return icon(gtx, "eyeoff", 20, col)
								}),
								layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									l := material.Label(u.th, 14, "Sekali lihat")
									l.Color = u.t.Text
									return l.Layout(gtx)
								}),
								layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{Size: gtx.Constraints.Min} }),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									s := "Mati"
									col := u.t.Text2
									if vo {
										s, col = "Aktif", u.t.Accent
									}
									l := material.Label(u.th, 13, s)
									l.Color = col
									return l.Layout(gtx)
								}),
							)
						})
					}),
				)
			}
			children = append(children,
				layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{Size: gtx.Constraints.Min} }),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return u.pendCancel.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									l := material.Label(u.th, 14.5, "Batal")
									l.Color = u.t.Text2
									return l.Layout(gtx)
								})
							})
						}),
						layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return u.pendSend.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									l := material.Label(u.th, 14.5, "Kirim")
									l.Color, l.Font.Weight = u.t.Accent, font.Medium
									return l.Layout(gtx)
								})
							})
						}),
					)
				}),
			)
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
		})
		call := macro.Stop()
		rr := gtx.Dp(12)
		paint.FillShape(gtx.Ops, u.t.Bg, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: rr, NE: rr, SE: rr, SW: rr}.Op(gtx.Ops))
		call.Add(gtx.Ops)
		return dims
	})
}

// groupEditLayer — modal edit info grup: editor Nama + Deskripsi + Simpan/Batal
// → core.SetGroupSubject + SetGroupDescription.
func (u *UI) groupEditLayer(gtx layout.Context) layout.Dimensions {
	paint.FillShape(gtx.Ops, color.NRGBA{A: 130}, clip.Rect{Max: gtx.Constraints.Max}.Op())
	for u.gedCancel.Clicked(gtx) {
		u.overlay = ""
	}
	for u.gedSave.Clicked(gtx) {
		if u.core != nil && u.selected != "" {
			if n := strings.TrimSpace(u.gedName.Text()); n != "" {
				u.core.SetGroupSubject(u.selected, n)
			}
			u.core.SetGroupDescription(u.selected, strings.TrimSpace(u.gedDesc.Text()))
			u.selName = strings.TrimSpace(u.gedName.Text())
		}
		u.overlay = ""
	}
	field := func(gtx layout.Context, ed *widget.Editor, hint string) layout.Dimensions {
		macro := op.Record(gtx.Ops)
		dims := layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			e := material.Editor(u.th, ed, hint)
			e.Color, e.HintColor, e.TextSize = u.t.Text, u.t.Text2, unit.Sp(15)
			return e.Layout(gtx)
		})
		call := macro.Stop()
		r := gtx.Dp(8)
		paint.FillShape(gtx.Ops, u.t.SearchBg, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
		call.Add(gtx.Ops)
		return dims
	}
	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		w := gtx.Dp(340)
		gtx.Constraints.Min.X, gtx.Constraints.Max.X = w, w
		macro := op.Record(gtx.Ops)
		dims := layout.UniformInset(unit.Dp(18)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					l := material.Label(u.th, 17, "Edit info grup")
					l.Color, l.Font.Weight = u.t.Text, font.Medium
					return l.Layout(gtx)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions { return field(gtx, &u.gedName, "Nama grup") }),
				layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions { return field(gtx, &u.gedDesc, "Deskripsi") }),
				layout.Rigid(layout.Spacer{Height: unit.Dp(14)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{Size: gtx.Constraints.Min} }),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return u.gedCancel.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									l := material.Label(u.th, 14.5, "Batal")
									l.Color = u.t.Text2
									return l.Layout(gtx)
								})
							})
						}),
						layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return u.gedSave.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									l := material.Label(u.th, 14.5, "Simpan")
									l.Color, l.Font.Weight = u.t.Accent, font.Medium
									return l.Layout(gtx)
								})
							})
						}),
					)
				}),
			)
		})
		call := macro.Stop()
		rr := gtx.Dp(12)
		paint.FillShape(gtx.Ops, u.t.Bg, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: rr, NE: rr, SE: rr, SW: rr}.Op(gtx.Ops))
		call.Add(gtx.Ops)
		return dims
	})
}

// inviteLinkLayer — modal link undangan grup: tampil link (atau "Memuat…") +
// tombol Salin (clipboard) + Tutup.
func (u *UI) inviteLinkLayer(gtx layout.Context) layout.Dimensions {
	paint.FillShape(gtx.Ops, color.NRGBA{A: 130}, clip.Rect{Max: gtx.Constraints.Max}.Op())
	link := u.inviteLink
	shown := link
	if shown == "" {
		shown = "Memuat…"
	}
	for u.inviteClose.Clicked(gtx) {
		u.overlay = ""
	}
	for u.inviteCopy.Clicked(gtx) {
		if link != "" {
			gtx.Execute(clipboard.WriteCmd{Type: "application/text", Data: io.NopCloser(strings.NewReader(link))})
		}
	}
	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		w := gtx.Dp(340)
		gtx.Constraints.Min.X, gtx.Constraints.Max.X = w, w
		macro := op.Record(gtx.Ops)
		dims := layout.UniformInset(unit.Dp(18)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					l := material.Label(u.th, 17, "Link undangan grup")
					l.Color, l.Font.Weight = u.t.Text, font.Medium
					return l.Layout(gtx)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					l := material.Label(u.th, 13.5, shown)
					l.Color, l.MaxLines = u.t.Accent, 2
					return l.Layout(gtx)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(14)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{Size: gtx.Constraints.Min} }),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return u.inviteClose.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									l := material.Label(u.th, 14.5, "Tutup")
									l.Color = u.t.Text2
									return l.Layout(gtx)
								})
							})
						}),
						layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return u.inviteCopy.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									l := material.Label(u.th, 14.5, "Salin")
									l.Color, l.Font.Weight = u.t.Accent, font.Medium
									return l.Layout(gtx)
								})
							})
						}),
					)
				}),
			)
		})
		call := macro.Stop()
		rr := gtx.Dp(12)
		paint.FillShape(gtx.Ops, u.t.Bg, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: rr, NE: rr, SE: rr, SW: rr}.Op(gtx.Ops))
		call.Add(gtx.Ops)
		return dims
	})
}

// scheduleLayer — modal jadwalkan pesan: pratinjau teks + 3 preset waktu + batal.
// Pilih preset → core.ScheduleMessage(chat, teks, unix) lalu kosongkan composer.
func (u *UI) scheduleLayer(gtx layout.Context) layout.Dimensions {
	paint.FillShape(gtx.Ops, color.NRGBA{A: 130}, clip.Rect{Max: gtx.Constraints.Max}.Op())
	now := time.Now()
	in1h := now.Add(time.Hour)
	tonight := time.Date(now.Year(), now.Month(), now.Day(), 20, 0, 0, 0, now.Location())
	if !tonight.After(now) {
		tonight = tonight.AddDate(0, 0, 1)
	}
	tomorrow := time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, now.Location()).AddDate(0, 0, 1)
	presets := []struct {
		label string
		at    time.Time
	}{
		{"1 jam lagi (" + in1h.Format("15.04") + ")", in1h},
		{"Malam ini, " + tonight.Format("15.04"), tonight},
		{"Besok pagi, " + tomorrow.Format("15.04"), tomorrow},
	}
	for i := range u.schedItems {
		for u.schedItems[i].Clicked(gtx) {
			txt := strings.TrimSpace(u.editor.Text())
			if u.core != nil && u.selected != "" && txt != "" {
				u.core.ScheduleMessage(u.selected, txt, presets[i].at.Unix())
			}
			u.editor.SetText("")
			delete(u.drafts, u.selected)
			u.overlay = ""
		}
	}
	for u.schedCancel.Clicked(gtx) {
		u.overlay = ""
	}
	preview := strings.TrimSpace(u.editor.Text())

	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		w := gtx.Dp(320)
		gtx.Constraints.Min.X, gtx.Constraints.Max.X = w, w
		macro := op.Record(gtx.Ops)
		dims := layout.UniformInset(unit.Dp(18)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			children := []layout.FlexChild{
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					l := material.Label(u.th, 17, "Jadwalkan pesan")
					l.Color, l.Font.Weight = u.t.Text, font.Medium
					return l.Layout(gtx)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					l := material.Label(u.th, 13.5, preview)
					l.Color, l.MaxLines = u.t.Text2, 2
					return l.Layout(gtx)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
			}
			for i := range presets {
				i := i
				children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return u.schedItems[i].Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{Top: unit.Dp(10), Bottom: unit.Dp(10), Left: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints.Min.X = gtx.Constraints.Max.X
							return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return layout.Inset{Right: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
										return icon(gtx, "clock", 18, u.t.Accent)
									})
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									l := material.Label(u.th, 14.5, presets[i].label)
									l.Color = u.t.Text
									return l.Layout(gtx)
								}),
							)
						})
					})
				}))
			}
			children = append(children,
				layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.E.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return u.schedCancel.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								l := material.Label(u.th, 14.5, "Batal")
								l.Color = u.t.Text2
								return l.Layout(gtx)
							})
						})
					})
				}),
			)
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
		})
		call := macro.Stop()
		rr := gtx.Dp(12)
		paint.FillShape(gtx.Ops, u.t.Bg, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: rr, NE: rr, SE: rr, SW: rr}.Op(gtx.Ops))
		call.Add(gtx.Ops)
		return dims
	})
}

// pinnedBarView — bilah pesan-tersemat di bawah header (ikon pin accent + teks
// pesan + jumlah bila >1). Ketuk → lompat ke pesan tersemat terbaru.
func (u *UI) pinnedBarView(gtx layout.Context, pinned []app.MessageDTO) layout.Dimensions {
	h := gtx.Dp(40)
	w := gtx.Constraints.Max.X
	sz := image.Pt(w, h)
	paint.FillShape(gtx.Ops, u.t.HeadBg, clip.Rect{Max: sz}.Op())
	paint.FillShape(gtx.Ops, u.t.Divider, clip.Rect{Min: image.Pt(0, h-1), Max: sz}.Op())

	cur := u.pinnedIdx
	if cur >= len(pinned) {
		cur = 0
	}
	// indikator segmen kiri (satu segmen per pin, segmen aktif terang) — ala WhatsApp.
	bw := gtx.Dp(3)
	if n := len(pinned); n > 1 {
		gap := gtx.Dp(2)
		pad := gtx.Dp(6)
		seg := (h - 2*pad - (n-1)*gap) / n
		if seg < gtx.Dp(3) {
			seg = gtx.Dp(3)
		}
		y := pad
		for i := 0; i < n; i++ {
			c := withAlpha(u.t.Accent, 0x55)
			if i == cur {
				c = u.t.Accent
			}
			paint.FillShape(gtx.Ops, c, clip.RRect{Rect: image.Rect(0, y, bw, y+seg), NW: bw / 2, NE: bw / 2, SE: bw / 2, SW: bw / 2}.Op(gtx.Ops))
			y += seg + gap
		}
	} else {
		paint.FillShape(gtx.Ops, u.t.Accent, clip.Rect{Max: image.Pt(bw, h)}.Op())
	}

	top := pinned[cur]
	preview := top.Text
	if preview == "" && top.Type != "" {
		preview = "[" + top.Type + "]"
	}
	label := "Pesan tersemat"
	if len(pinned) > 1 {
		label = "Pesan tersemat " + itoa(cur+1) + "/" + itoa(len(pinned))
	}
	gtx.Constraints.Min, gtx.Constraints.Max = sz, sz
	return u.pinnedBar.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Left: unit.Dp(14), Right: unit.Dp(12), Top: unit.Dp(5), Bottom: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Right: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return icon(gtx, "pin", 16, u.t.Accent)
					})
				}),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							l := material.Label(u.th, 11.5, label)
							l.Color = u.t.Accent
							l.MaxLines = 1
							return l.Layout(gtx)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							l := material.Label(u.th, 13, preview)
							l.Color = u.t.Text2
							l.MaxLines = 1
							return l.Layout(gtx)
						}),
					)
				}),
			)
		})
	})
}

// handleMsgExtras — proses klik "Baca selengkapnya/Sembunyikan" DI LUAR layout
// list (Clicked di dalam closure material.List tak selalu memicu).
func (u *UI) handleMsgExtras(gtx layout.Context) {
	for id, c := range u.moreClicks {
		for c.Clicked(gtx) {
			u.expanded[id] = !u.expanded[id]
		}
	}
}

func (u *UI) conversation(gtx layout.Context) layout.Dimensions {
	drawWallpaper(gtx, u.t)
	if u.view != "channels" {
		u.openChannel = "" // keluar dari pane channels → tutup reader
	}
	if u.view == "channels" && u.openChannel != "" {
		return u.channelReader(gtx) // pembaca post channel
	}
	if u.selected == "" {
		return EmptyConversationView(gtx, u.th, u.t) // splash saja (tanpa pil demo)
	}
	u.syncDraft()          // ganti chat → simpan draft lama, muat draft chat baru
	u.maybeLoadOlder()     // gulir mendekati atas → minta history lama (lazy, throttled)
	u.handleMsgExtras(gtx) // toggle "Baca selengkapnya" (DI LUAR list, ala handleChatClicks)
	pinned := u.pinnedMsgs()
	if u.pinnedIdx >= len(pinned) {
		u.pinnedIdx = 0
	}
	for u.pinnedBar.Clicked(gtx) { // ketuk bar tersemat → lompat + siklus ke pin berikut
		if len(pinned) > 0 {
			u.jumpToMessage(pinned[u.pinnedIdx].ID)
			u.pinnedIdx = (u.pinnedIdx + 1) % len(pinned)
		}
	}
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return u.convHeader(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if len(pinned) == 0 {
				return layout.Dimensions{}
			}
			return u.pinnedBarView(gtx, pinned)
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			if len(u.msgClicks) < len(u.messages) { // jamin sebelum index (mid-frame GetMessages)
				u.msgClicks = make([]widget.Clickable, len(u.messages))
			}
			if len(u.quoteClicks) < len(u.messages) {
				u.quoteClicks = make([]widget.Clickable, len(u.messages))
			}
			return layout.Stack{}.Layout(gtx,
				layout.Expanded(func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return material.List(u.th, &u.msgList).Layout(gtx, len(u.messages), func(gtx layout.Context, i int) layout.Dimensions {
							return u.bubble(gtx, i)
						})
					})
				}),
				layout.Stacked(func(gtx layout.Context) layout.Dimensions {
					return u.scrollFab(gtx)
				}),
			)
		}),
		// bubble "sedang mengetik" (titik bergerak) di atas composer bila peer mengetik.
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if u.typingOf(u.selected) == "" {
				return layout.Dimensions{}
			}
			return u.typingBubble(gtx)
		}),
		// saran @mention (grup) di atas composer saat mengetik "@…".
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return u.mentionSuggest(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return u.composer(gtx)
		}),
	)
}

// typingBubble — bubble kecil ala pesan masuk berisi 3 titik beranimasi (bouncing).
// Animasi via InvalidateCmd periodik; fase titik dari gtx.Now.
func (u *UI) typingBubble(gtx layout.Context) layout.Dimensions {
	gtx.Execute(op.InvalidateCmd{At: gtx.Now.Add(280 * time.Millisecond)}) // redraw → animasi
	phase := 0
	if !gtx.Now.IsZero() {
		phase = int(gtx.Now.UnixNano()/int64(280*time.Millisecond)) % 3
	}
	// grup → avatar pengirim di kiri bubble (ala WhatsApp: tahu siapa yg mengetik).
	whoName, whoJID := u.typingWhoOf(u.selected)
	bub := func(gtx layout.Context) layout.Dimensions {
		macro := op.Record(gtx.Ops)
		dims := layout.Inset{Top: unit.Dp(11), Bottom: unit.Dp(11), Left: unit.Dp(14), Right: unit.Dp(14)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			children := make([]layout.FlexChild, 0, 5)
			for i := 0; i < 3; i++ {
				if i > 0 {
					children = append(children, layout.Rigid(layout.Spacer{Width: unit.Dp(5)}.Layout))
				}
				dimDot := i != phase // titik aktif lebih terang
				children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					d := gtx.Dp(7)
					col := u.t.Text2
					if dimDot {
						col.A = 0x66
					}
					paint.FillShape(gtx.Ops, col, clip.Ellipse{Max: image.Pt(d, d)}.Op(gtx.Ops))
					return layout.Dimensions{Size: image.Pt(d, d)}
				}))
			}
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx, children...)
		})
		call := macro.Stop()
		r := gtx.Dp(14)
		// bubble masuk (InBg), sudut kiri-atas kecil (ekor) ala WhatsApp.
		paint.FillShape(gtx.Ops, u.t.InBg, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: gtx.Dp(4), NE: r, SE: r, SW: r}.Op(gtx.Ops))
		call.Add(gtx.Ops)
		return dims
	}
	return layout.Inset{Left: unit.Dp(14), Top: unit.Dp(2), Bottom: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		if whoJID == "" { // DM → bubble saja (tanpa avatar)
			return bub(gtx)
		}
		// grup → avatar kecil + bubble berdampingan (Bottom-align ke ekor).
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.End}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return u.avatar(gtx, whoName, whoJID, 26)
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout),
			layout.Rigid(bub),
		)
	})
}

// inChatSearchHeader — bilah cari-dalam-chat (ganti header): ← kembali + input +
// "n/total" + navigasi naik/turun. Cocokkan teks pesan chat aktif; nav → lompat +
// sorot. Submit (Enter) = lompat ke kecocokan berikutnya.
func (u *UI) inChatSearchHeader(gtx layout.Context) layout.Dimensions {
	h := gtx.Dp(60)
	sz := image.Pt(gtx.Constraints.Max.X, h)
	paint.FillShape(gtx.Ops, u.t.HeadBg, clip.Rect{Max: sz}.Op())
	paint.FillShape(gtx.Ops, u.t.Divider, clip.Rect{Min: image.Pt(0, h-1), Max: sz}.Op())

	// hitung kecocokan (teks pesan mengandung query, case-insensitive).
	q := strings.ToLower(strings.TrimSpace(u.inChatEd.Text()))
	u.inChatMatch = u.inChatMatch[:0]
	if q != "" {
		for i := range u.messages {
			if strings.Contains(strings.ToLower(u.messages[i].Text), q) {
				u.inChatMatch = append(u.inChatMatch, i)
			}
		}
	}
	if u.inChatCur >= len(u.inChatMatch) {
		u.inChatCur = 0
	}
	jump := func() {
		if len(u.inChatMatch) > 0 && u.inChatCur < len(u.inChatMatch) {
			u.jumpToMessage(u.messages[u.inChatMatch[u.inChatCur]].ID)
		}
	}
	for u.inChatBack.Clicked(gtx) {
		u.inChatSearch = false
		u.inChatEd.SetText("")
		u.inChatMatch, u.inChatCur = nil, 0
	}
	for u.inChatNext.Clicked(gtx) {
		if n := len(u.inChatMatch); n > 0 {
			u.inChatCur = (u.inChatCur + 1) % n
			jump()
		}
	}
	for u.inChatPrev.Clicked(gtx) {
		if n := len(u.inChatMatch); n > 0 {
			u.inChatCur = (u.inChatCur - 1 + n) % n
			jump()
		}
	}
	for { // Enter → kecocokan berikutnya
		ev, ok := u.inChatEd.Update(gtx)
		if !ok {
			break
		}
		if _, ok := ev.(widget.SubmitEvent); ok {
			if n := len(u.inChatMatch); n > 0 {
				u.inChatCur = (u.inChatCur + 1) % n
				jump()
			}
		}
	}

	counter := ""
	switch {
	case q == "":
	case len(u.inChatMatch) == 0:
		counter = "0"
	default:
		counter = itoa(u.inChatCur+1) + "/" + itoa(len(u.inChatMatch))
	}

	gtx.Constraints.Min, gtx.Constraints.Max = sz, sz
	return layout.Inset{Left: unit.Dp(6), Right: unit.Dp(10), Top: unit.Dp(10), Bottom: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { return u.glyphBtn(gtx, &u.inChatBack, "back") }),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				e := material.Editor(u.th, &u.inChatEd, "Cari dalam chat")
				e.Color, e.HintColor, e.TextSize = u.t.Text, u.t.Text2, unit.Sp(15)
				return e.Layout(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if counter == "" {
					return layout.Dimensions{}
				}
				return layout.Inset{Right: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					l := material.Label(u.th, 13, counter)
					l.Color = u.t.Text2
					l.MaxLines = 1
					return l.Layout(gtx)
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { return u.glyphBtn(gtx, &u.inChatPrev, "chevronup") }),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { return u.glyphBtn(gtx, &u.inChatNext, "chevrondown") }),
		)
	})
}

// selectionHeader — toolbar mode-pilih: ✕ batal + "N dipilih" + teruskan + hapus.
func (u *UI) selectionHeader(gtx layout.Context) layout.Dimensions {
	h := gtx.Dp(60)
	sz := image.Pt(gtx.Constraints.Max.X, h)
	paint.FillShape(gtx.Ops, u.t.HeadBg, clip.Rect{Max: sz}.Op())
	paint.FillShape(gtx.Ops, u.t.Divider, clip.Rect{Min: image.Pt(0, h-1), Max: sz}.Op())
	for u.selCancel.Clicked(gtx) {
		u.exitSel()
	}
	for u.selDelete.Clicked(gtx) {
		u.deleteSelected()
	}
	for u.selFwd.Clicked(gtx) {
		u.forwardSelected()
	}
	gtx.Constraints.Min, gtx.Constraints.Max = sz, sz
	return layout.Inset{Left: unit.Dp(6), Right: unit.Dp(10), Top: unit.Dp(10), Bottom: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { return u.glyphBtn(gtx, &u.selCancel, "close") }),
			layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				lbl := material.Label(u.th, 17, itoa(len(u.selSet))+" dipilih")
				lbl.Color = u.t.Text
				lbl.Font.Weight = font.Medium
				lbl.MaxLines = 1
				return lbl.Layout(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { return u.glyphBtn(gtx, &u.selFwd, "forward") }),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { return u.glyphBtn(gtx, &u.selDelete, "trash") }),
		)
	})
}

func (u *UI) convHeader(gtx layout.Context) layout.Dimensions {
	if u.selMode {
		return u.selectionHeader(gtx)
	}
	if u.inChatSearch {
		return u.inChatSearchHeader(gtx)
	}
	for u.headerClick.Clicked(gtx) {
		u.overlay = "info" // klik header → info drawer
	}
	h := gtx.Dp(60)
	sz := image.Pt(gtx.Constraints.Max.X, h)
	u.headerClick.Layout(gtx, func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{Size: sz} })
	paint.FillShape(gtx.Ops, u.t.HeadBg, clip.Rect{Max: sz}.Op())
	paint.FillShape(gtx.Ops, u.t.Divider, clip.Rect{Min: image.Pt(0, h-1), Max: sz}.Op())
	gtx.Constraints.Min, gtx.Constraints.Max = sz, sz
	// ikon overflow header → menu aksi chat yg sedang terbuka (reuse chatctx).
	for u.headMenuClick.Clicked(gtx) {
		for i := range u.chats {
			if u.chats[i].ID == u.selected {
				u.chatCtxChat = u.chats[i] // snapshot chat terbuka
				u.overlay = "chatctx"
				break
			}
		}
	}
	for u.headSearchClick.Clicked(gtx) { // ikon cari → cari DALAM chat aktif
		u.inChatSearch = true
		u.inChatEd.SetText("")
		u.inChatMatch, u.inChatCur = nil, 0
	}
	// ikon aksi (telepon/cari/overflow) dipatok MUTLAK di kanan — Flexed(1) tak
	// melebar andal di sini (sama spt titlebar). avatar+nama di kiri [18..btnsX].
	btnW := gtx.Dp(40)
	rpad := gtx.Dp(8)
	hpad := gtx.Dp(18)
	btnsX := sz.X - rpad - 4*btnW // 4 ikon: video, telepon, cari, overflow
	if btnsX < hpad {
		btnsX = hpad
	}
	// kiri: avatar + nama/subtitle (terpusat vertikal via offset).
	avD := gtx.Dp(40)
	ly := (h - avD) / 2
	lgtx := gtx
	lgtx.Constraints.Min, lgtx.Constraints.Max = image.Pt(0, 0), image.Pt(btnsX-hpad, h)
	lo := op.Offset(image.Pt(hpad, ly)).Push(gtx.Ops)
	layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(lgtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions { return u.avatar(gtx, u.selName, u.selected, 40) }),
		layout.Rigid(layout.Spacer{Width: unit.Dp(13)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					lbl := material.Label(u.th, 16, u.selName)
					lbl.Color = u.t.Text
					lbl.Font.Weight = font.Medium
					lbl.MaxLines = 1
					return lbl.Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if u.subtitle == "" {
						return layout.Dimensions{}
					}
					col := u.t.Text2 // presence → text2; mengetik/merekam → accent
					if strings.Contains(u.subtitle, "mengetik") || strings.Contains(u.subtitle, "merekam") {
						col = u.t.Accent
					}
					lbl := material.Label(u.th, 12.5, u.subtitle)
					lbl.Color = col
					lbl.MaxLines = 1
					return lbl.Layout(gtx)
				}),
			)
		}),
	)
	lo.Pop()

	// kanan: 3 ikon aksi dipatok mutlak, terpusat vertikal.
	by := (h - btnW) / 2
	acts := []struct {
		c  *widget.Clickable
		ic string
	}{{nil, "video"}, {nil, "calls"}, {&u.headSearchClick, "search"}, {&u.headMenuClick, "overflow"}}
	for i, a := range acts {
		o := op.Offset(image.Pt(btnsX+i*btnW, by)).Push(gtx.Ops)
		u.glyphBtn(gtx, a.c, a.ic)
		o.Pop()
	}
	return layout.Dimensions{Size: sz}
}

// ---- bubble pesan (.bubble: in/out, RRect, ekor) ----
func (u *UI) bubble(gtx layout.Context, idx int) layout.Dimensions {
	if idx < 0 || idx >= len(u.messages) { // u.messages bisa menyusut mid-frame (refresh)
		return layout.Dimensions{}
	}
	m := u.messages[idx]
	if idx < len(u.msgClicks) {
		for u.msgClicks[idx].Clicked(gtx) {
			switch {
			case u.selMode:
				u.toggleSel(m.ID) // mode pilih → tap = pilih/lepas
			case m.Type == "voice" && u.OnPlayVoice != nil:
				u.OnPlayVoice(u.selected, m.ID) // tap voice → putar
			case (m.Type == "video" || m.Type == "gif") && u.OnPlayVideo != nil:
				u.OnPlayVideo(u.selected, m.ID, m.Type) // tap video/gif → putar
			case m.Type == "image":
				u.lightboxMsg, u.lightboxCap = m.ID, m.Text // tap gambar → lightbox layar penuh
				u.overlay = "lightbox"
			default:
				u.ctxIdx, u.ctxMsg = idx, m // snapshot pesan (index bisa bergeser)
				u.overlay = "msgctx"        // klik pesan → context-menu
			}
		}
	}
	out := m.Dir == "out"
	bg := u.t.InBg
	if out {
		bg = u.t.OutBg
	}
	groupIn := u.selGroup && m.Dir == "in"

	// grouping (ala WhatsApp/IG): pesan beruntun dari pengirim sama → rapatkan jarak,
	// ekor hanya di bubble PERTAMA run, nama pengirim (grup) hanya di pertama.
	sameRun := func(a, b app.MessageDTO) bool {
		return a.Dir == b.Dir && a.Sender == b.Sender && !a.Revoked && !b.Revoked &&
			a.Type != "sticker" && b.Type != "sticker"
	}
	firstOfRun := idx == 0 || !sameRun(u.messages[idx-1], m)

	// lebar bubble: 66% area, TAPI dibatasi absolut ~560dp (jendela lebar tak bikin
	// baris super-panjang) + sisakan ruang avatar di grup.
	avail := gtx.Constraints.Max.X
	if u.selGroup {
		avail -= gtx.Dp(46) // gutter avatar
	}
	maxW := avail * 66 / 100
	if cap := gtx.Dp(560); maxW > cap {
		maxW = cap
	}
	if min := gtx.Dp(90); maxW < min {
		maxW = min
	}
	// susun konten bubble
	content := func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Max.X = maxW
		return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(13), Right: unit.Dp(13)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			// metaRow — jam + "diedit" + centang status (dipakai inline ATAU baris bawah).
			metaRow := func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if !m.Edited || m.Revoked {
							return layout.Dimensions{}
						}
						lbl := material.Label(u.th, 11, "Diedit  ")
						lbl.Color, lbl.Font.Style = u.t.Text2, font.Italic
						return lbl.Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						lbl := material.Label(u.th, 11, m.Time)
						lbl.Color = u.t.Text2
						return lbl.Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if !out {
							return layout.Dimensions{}
						}
						return layout.Inset{Left: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return u.statusTick(gtx, m.Status)
						})
					}),
				)
			}
			var inlineMeta bool // diset di body teks pendek → meta ikut di baris yg sama
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if !groupIn || m.Sender == "" || !firstOfRun {
						return layout.Dimensions{} // nama pengirim hanya di bubble pertama run
					}
					lbl := material.Label(u.th, 13, m.Sender)
					lbl.Color = avatarColor(m.Sender)
					lbl.Font.Weight = font.Bold
					return layout.Inset{Bottom: unit.Dp(3)}.Layout(gtx, lbl.Layout) // jarak nama→isi
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if m.QuoteName == "" && m.QuoteText == "" {
						return layout.Dimensions{}
					}
					qb := func(gtx layout.Context) layout.Dimensions { return u.quoteBlock(gtx, m, out) }
					if m.QuoteID != "" && idx < len(u.quoteClicks) { // ketuk → lompat ke pesan asal
						for u.quoteClicks[idx].Clicked(gtx) {
							u.jumpToMessage(m.QuoteID)
						}
						return u.quoteClicks[idx].Layout(gtx, qb)
					}
					return qb(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if m.Revoked { // pesan ditarik pengirim → placeholder miring + ikon
						return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions { return icon(gtx, "block", 15, u.t.Text2) }),
							layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								lbl := material.Label(u.th, 15, "Pesan ini telah dihapus")
								lbl.Color = u.t.Text2
								lbl.Font.Style = font.Italic
								return lbl.Layout(gtx)
							}),
						)
					}
					// media bubble HUG-content: batasi lebar per-tipe (jangan melar ke 66%).
					capW := func(dp int) {
						if c := gtx.Dp(unit.Dp(dp)); c < gtx.Constraints.Max.X {
							gtx.Constraints.Min.X, gtx.Constraints.Max.X = c, c
						}
					}
					switch m.Type {
					case "image", "video", "gif":
						if m.Text == "" {
							inlineMeta = true // waktu jadi chip overlay di thumbnail → tak ada baris meta bawah
						}
						return u.mediaThumb(gtx, m) // thumbnail + caption (lebar diatur di dalam)
					case "sticker":
						return u.stickerBubble(gtx, m) // transparan, tanpa gelembung
					case "poll":
						capW(300)
						return u.pollBubble(gtx, m) // pertanyaan + opsi
					case "document":
						capW(300)
						return u.docBubble(gtx, m)
					case "audio":
						capW(280)
						return u.musicBubble(gtx, m) // berkas musik: art + judul + progress
					case "voice", "ptt":
						capW(260)
						return u.voiceBubble(gtx, m)
					case "location":
						capW(270)
						return u.locationBubble(gtx, m)
					case "contact", "vcard":
						capW(260)
						return u.contactBubble(gtx, m)
					}
					txt := m.Text
					if txt == "" && m.Type != "" && m.Type != "text" {
						txt = "[" + m.Type + "]"
					}
					// teks panjang → ringkas + "Baca selengkapnya" (show more/less).
					runes := []rune(txt)
					nl := strings.Count(txt, "\n")
					isLong := (m.Type == "" || m.Type == "text") && !m.Revoked && (len(runes) > 450 || nl > 7)
					expanded := u.expanded[m.ID]
					disp := txt
					if isLong && !expanded {
						cut := string(runes) // potong: maks 6 baris ATAU 380 char, mana lebih dulu
						if parts := strings.SplitN(cut, "\n", 7); len(parts) > 6 {
							cut = strings.Join(parts[:6], "\n")
						}
						if cr := []rune(cut); len(cr) > 380 {
							cut = string(cr[:380])
						}
						disp = strings.TrimRight(cut, " \n\t") + "…"
					}
					textW := func(gtx layout.Context) layout.Dimensions {
						body := func(gtx layout.Context) layout.Dimensions {
							if n := emojiOnlyCount(txt); n > 0 { // 1-3 emoji saja → diperbesar (ala WA), tanpa format
								sz := unit.Sp(26)
								if n == 1 {
									sz = unit.Sp(40)
								} else if n == 2 {
									sz = unit.Sp(32)
								}
								lbl := material.Label(u.th, sz, txt)
								lbl.Color = u.t.Text
								return lbl.Layout(gtx)
							}
							// format WhatsApp (*tebal* _miring_ ~coret~ ```mono```) + URL biru + @mention.
							return u.formattedText(gtx, disp, m.Mentions, unit.Sp(15), m.ID)
						}
						if !isLong {
							return body(gtx)
						}
						label := "Baca selengkapnya"
						if expanded {
							label = "Sembunyikan"
						}
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
							layout.Rigid(body),
							layout.Rigid(layout.Spacer{Height: unit.Dp(3)}.Layout),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return u.clk(u.moreClicks, m.ID).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									l := material.Label(u.th, 13.5, label)
									l.Color, l.Font.Weight = u.t.Text2, font.Medium
									return l.Layout(gtx)
								})
							}),
						)
					}
					var card layout.Widget
					if url := firstURL(txt); url != "" { // pratinjau tautan di atas teks
						u.ensureLinkPreview(url)
						card = u.linkCardWidget(url)
					}
					trBlock := u.translatedWidget(m.ID) // terjemahan di bawah teks
					if card == nil && trBlock == nil {
						// teks 1-baris pendek → jam INLINE di kanan baris (ala WhatsApp).
						if (m.Type == "" || m.Type == "text") && !m.Revoked {
							cg := gtx
							cg.Constraints.Min = image.Point{}
							tmac := op.Record(gtx.Ops)
							td := textW(cg) // teks di-wrap pd lebar bubble
							tmac.Stop()
							// tinggi 1-baris SEBENARNYA: ukur teks tanpa wrap (lebar tak hingga).
							wg := gtx
							wg.Constraints = layout.Constraints{Max: image.Pt(1<<20, 1<<20)}
							lmac := op.Record(gtx.Ops)
							sd := textW(wg)
							lmac.Stop()
							mmac := op.Record(gtx.Ops)
							md := metaRow(cg)
							mmac.Stop()
							gap := gtx.Dp(8)
							// 1 baris (tinggi wrap == tinggi tanpa-wrap) & teks+meta muat → inline kanan.
							if td.Size.Y <= sd.Size.Y && td.Size.X+gap+md.Size.X <= gtx.Constraints.Max.X {
								inlineMeta = true
								return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Baseline}.Layout(gtx,
									layout.Rigid(textW),
									layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
									layout.Rigid(metaRow),
								)
							}
						}
						return textW(gtx)
					}
					children := make([]layout.FlexChild, 0, 4)
					if card != nil {
						children = append(children, layout.Rigid(card), layout.Rigid(layout.Spacer{Height: unit.Dp(5)}.Layout))
					}
					children = append(children, layout.Rigid(textW))
					if trBlock != nil {
						children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout), layout.Rigid(trBlock))
					}
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if inlineMeta { // sudah ditaruh inline di baris teks → tak ada baris meta bawah
						return layout.Dimensions{}
					}
					// .meta: jam + centang status, rata kanan-bawah.
					return layout.E.Layout(gtx, metaRow)
				}),
			)
		})
	}
	// bubble dgn latar RRect + alignment in/out
	align := layout.W
	if out {
		align = layout.E
	}
	// stiker & emoji-only → TANPA gelembung (transparan), ala WhatsApp/IG.
	noBubble := m.Type == "sticker" || (m.Revoked == false && (m.Type == "" || m.Type == "text") && emojiOnlyCount(m.Text) > 0)
	bubbleBody := func(gtx layout.Context) layout.Dimensions {
		// rekam konten utk ukur, lalu gambar bg di belakang
		macro := op.Record(gtx.Ops)
		dims := content(gtx)
		call := macro.Stop()
		if noBubble { // tanpa latar gelembung
			call.Add(gtx.Ops)
			return dims
		}
		r := gtx.Dp(14)   // radius modern (lebih lembut dari 18)
		tail := gtx.Dp(4) // sudut "ekor" dekat pengirim — HANYA di bubble pertama run
		tl, tr := r, r
		if firstOfRun {
			if out {
				tr = tail
			} else {
				tl = tail
			}
		}
		rr := clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: tl, NE: tr, SE: r, SW: r}
		paint.FillShape(gtx.Ops, bg, rr.Op(gtx.Ops))
		// sorot sesaat saat dilompati dari kutipan (pudar ~1.6s), accent tipis di
		// belakang konten agar tetap terbaca.
		if u.hlMsg == m.ID {
			if el := time.Since(u.hlAt); el < 1600*time.Millisecond {
				hc := u.t.Accent
				hc.A = uint8(float64(80) * (1 - el.Seconds()/1.6))
				paint.FillShape(gtx.Ops, hc, rr.Op(gtx.Ops))
				gtx.Execute(op.InvalidateCmd{}) // animasikan pudar
			} else {
				u.hlMsg = ""
			}
		}
		call.Add(gtx.Ops)
		return dims
	}
	colAlign := layout.Start
	if out {
		colAlign = layout.End
	}
	// jarak antar-bubble: lebar antar pengirim beda (firstOfRun), rapat dlm satu run.
	gapTop := unit.Dp(2)
	if firstOfRun {
		gapTop = unit.Dp(8)
	}
	// avatar gutter (grup): foto pengirim di sisi pengirim, sejajar ATAS bubble.
	// Hanya di bubble PERTAMA run (avatar di samping pesan teratas, ala WhatsApp);
	// bubble lain dlm run sisakan gutter kosong agar sejajar. Berlaku in + out (saya).
	avGutter := func(gtx layout.Context) layout.Dimensions {
		d := gtx.Dp(28)
		if !firstOfRun || noBubble {
			return layout.Dimensions{Size: image.Pt(d, d)}
		}
		name, jid := m.Sender, m.SenderID
		if out {
			name, jid = u.profName, u.selfJID
			if name == "" {
				name = "Saya"
			}
		}
		return u.avatar(gtx, name, jid, 28)
	}
	bubbleCol := func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical, Alignment: colAlign}.Layout(gtx,
			// klik HANYA pada bubble (bukan ruang kosong baris) → ctx-menu/lightbox.
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if idx < len(u.msgClicks) {
					return u.msgClicks[idx].Layout(gtx, bubbleBody)
				}
				return bubbleBody(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return u.reactionPills(gtx, m)
			}),
		)
	}
	wrap := func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: gapTop, Bottom: unit.Dp(1)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			if !u.selGroup { // DM → tanpa avatar (spt sekarang)
				return align.Layout(gtx, bubbleCol)
			}
			return align.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				sp := layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout)
				if out { // pesan saya → avatar di kanan, sejajar atas
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Start}.Layout(gtx,
						layout.Rigid(bubbleCol), sp, layout.Rigid(avGutter))
				}
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Start}.Layout(gtx,
					layout.Rigid(avGutter), sp, layout.Rigid(bubbleCol))
			})
		})
	}
	// Pemisah hari di atas bubble bila ganti tanggal (atau pesan pertama). Bandingkan
	// dgn pesan SEBELUMNYA yg punya Ts>0 (pesan ditarik/sistem bisa Ts==0 → jangan
	// picu pemisah palsu di hari yg sama).
	needSep := false
	if m.Ts > 0 {
		prevDay := int64(-1)
		for j := idx - 1; j >= 0; j-- {
			if u.messages[j].Ts > 0 {
				prevDay = dayKey(u.messages[j].Ts)
				break
			}
		}
		if prevDay < 0 || prevDay != dayKey(m.Ts) {
			needSep = true
		}
	}
	showUnread := u.unreadDivID != "" && m.ID == u.unreadDivID // divider "belum dibaca" di atas pesan ini
	if !needSep && !showUnread {
		return u.msgRow(gtx, idx, wrap)
	}
	children := make([]layout.FlexChild, 0, 3)
	if needSep {
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions { return u.daySeparator(gtx, dayLabel(m.Ts)) }))
	}
	if showUnread {
		n := u.unreadDivCount
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return u.unreadDivider(gtx, n)
		}))
	}
	children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions { return u.msgRow(gtx, idx, wrap) }))
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
}

// msgRow — baris pesan; di mode-pilih & terpilih, beri pita accent tipis selebar
// baris di belakangnya (penanda seleksi).
func (u *UI) msgRow(gtx layout.Context, idx int, wrap layout.Widget) layout.Dimensions {
	// msgClicks kini membungkus HANYA bubble (di bubbleCol), jadi baris cuma menata.
	if !u.selMode || idx >= len(u.messages) || !u.selSet[u.messages[idx].ID] {
		return wrap(gtx)
	}
	macro := op.Record(gtx.Ops)
	dims := wrap(gtx)
	call := macro.Stop()
	tint := u.t.Accent
	tint.A = 32
	paint.FillShape(gtx.Ops, tint, clip.Rect{Max: image.Pt(gtx.Constraints.Max.X, dims.Size.Y)}.Op())
	call.Add(gtx.Ops)
	return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, dims.Size.Y)}
}

// unreadDivider — pemisah "N PESAN BELUM DIBACA" (pil accent lembut, lebar penuh).
func (u *UI) unreadDivider(gtx layout.Context, n int) layout.Dimensions {
	label := "PESAN BELUM DIBACA"
	if n > 0 {
		label = itoa(n) + " PESAN BELUM DIBACA"
	}
	return layout.Inset{Top: unit.Dp(6), Bottom: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		macro := op.Record(gtx.Ops)
		dims := layout.Inset{Top: unit.Dp(5), Bottom: unit.Dp(5)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				l := material.Label(u.th, 12, label)
				l.Color = u.t.Accent
				l.Font.Weight = font.Medium
				l.MaxLines = 1
				return l.Layout(gtx)
			})
		})
		call := macro.Stop()
		bg := u.t.Accent
		bg.A = 28 // pita accent sangat lembut selebar layar
		paint.FillShape(gtx.Ops, bg, clip.Rect{Max: dims.Size}.Op())
		call.Add(gtx.Ops)
		return dims
	})
}

// statusTick — centang status pesan keluar (✓ sent, ✓✓ delivered/read; biru=read).
func (u *UI) statusTick(gtx layout.Context, status string) layout.Dimensions {
	name := "check"
	col := u.t.Text2
	switch status {
	case "read":
		name, col = "checks", u.t.Tick // biru baca
	case "delivered":
		name = "checks"
	case "sent", "":
		name = "check"
	}
	return icon(gtx, name, 16, col)
}

// daySeparator — pil tanggal di tengah (paritas .day-divider).
func (u *UI) daySeparator(gtx layout.Context, label string) layout.Dimensions {
	return layout.Inset{Top: unit.Dp(10), Bottom: unit.Dp(10)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			macro := op.Record(gtx.Ops)
			dims := layout.Inset{Top: unit.Dp(5), Bottom: unit.Dp(5), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				lbl := material.Label(u.th, 12, label)
				lbl.Color = u.t.Text2
				return lbl.Layout(gtx)
			})
			call := macro.Stop()
			r := gtx.Dp(8)
			paint.FillShape(gtx.Ops, u.t.Bg2, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
			call.Add(gtx.Ops)
			return dims
		})
	})
}

// dayKey — kunci hari (tahun-bulan-tanggal) dari unix detik utk bandingkan tanggal.
func dayKey(ts int64) int64 {
	if ts <= 0 {
		return 0
	}
	t := time.Unix(ts, 0)
	y, mo, d := t.Date()
	return int64(y)*10000 + int64(mo)*100 + int64(d)
}

// dayLabel — label pemisah: "Hari ini" / "Kemarin" / "2 Jan 2006" (Indonesia).
func dayLabel(ts int64) string {
	if ts <= 0 {
		return ""
	}
	t := time.Unix(ts, 0)
	now := time.Now()
	switch dayKey(ts) {
	case dayKey(now.Unix()):
		return "Hari ini"
	case dayKey(now.AddDate(0, 0, -1).Unix()):
		return "Kemarin"
	}
	bulan := []string{"", "Jan", "Feb", "Mar", "Apr", "Mei", "Jun", "Jul", "Agu", "Sep", "Okt", "Nov", "Des"}
	y, mo, d := t.Date()
	return itoa(d) + " " + bulan[int(mo)] + " " + itoa(y)
}

// clearReply membatalkan mode balas.
func (u *UI) clearReply() { u.replyTo, u.replyName, u.replyText = "", "", "" }

func (u *UI) composer(gtx layout.Context) layout.Dimensions {
	if u.recDemo || (u.core != nil && u.core.VoiceRecording()) {
		return u.recordingBar(gtx)
	}
	for u.emojiClick.Clicked(gtx) {
		u.reactMsgID = "" // tombol emoji composer → mode sisip (bukan reaksi pesan)
		u.overlay = "reaction"
	}
	for u.attachClick.Clicked(gtx) {
		u.overlay = "attach" // tombol "+" → menu lampiran
	}
	for u.stickerClick.Clicked(gtx) {
		u.overlay = "picker" // tombol stiker → picker stiker
	}
	for u.replyX.Clicked(gtx) {
		u.clearReply()
	}
	for u.editCancel.Clicked(gtx) { // batal edit → kosongkan composer
		u.clearEdit()
		u.editor.SetText("")
	}
	// tinggi composer DINAMIS: pil tumbuh hingga ~6 baris (Shift+Enter) → ukur kolom
	// dulu, lalu gambar latar HeadBg + garis atas setinggi itu.
	cgtx := gtx
	cgtx.Constraints.Min = image.Point{}
	macro := op.Record(gtx.Ops)
	dims := layout.Flex{Axis: layout.Vertical}.Layout(cgtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if u.editTarget != "" {
				return u.editBanner(gtx)
			}
			if u.replyTo == "" {
				return layout.Dimensions{}
			}
			return u.replyBanner(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Left: unit.Dp(16), Right: unit.Dp(16), Top: unit.Dp(11), Bottom: unit.Dp(11)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.End}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return u.glyphBtnTip(gtx, &u.emojiClick, "emoji", "Emoji & reaksi")
					}),
					layout.Rigid(layout.Spacer{Width: unit.Dp(4)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return u.glyphBtnTip(gtx, &u.stickerClick, "sticker", "Stiker & GIF")
					}),
					layout.Rigid(layout.Spacer{Width: unit.Dp(4)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return u.glyphBtnTip(gtx, &u.attachClick, "plus", "Lampirkan")
					}),
					layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return u.composerPill(gtx) }),
					layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions { return u.sendOrMic(gtx) }),
				)
			})
		}),
	)
	call := macro.Stop()
	sz := image.Pt(gtx.Constraints.Max.X, dims.Size.Y)
	paint.FillShape(gtx.Ops, u.t.HeadBg, clip.Rect{Max: sz}.Op())
	paint.FillShape(gtx.Ops, u.t.Divider, clip.Rect{Max: image.Pt(sz.X, 1)}.Op())
	call.Add(gtx.Ops)
	return layout.Dimensions{Size: sz}
}

// replyBanner — bilah kutipan di atas composer (garis accent + nama + teks + ✕).
func (u *UI) replyBanner(gtx layout.Context) layout.Dimensions {
	return layout.Inset{Left: unit.Dp(16), Right: unit.Dp(12), Top: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		macro := op.Record(gtx.Ops)
		dims := layout.Inset{Top: unit.Dp(6), Bottom: unit.Dp(6), Left: unit.Dp(10), Right: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							lbl := material.Label(u.th, 13, u.replyName)
							lbl.Color = u.t.Accent
							lbl.Font.Weight = font.Medium
							lbl.MaxLines = 1
							return lbl.Layout(gtx)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							lbl := material.Label(u.th, 13, u.replyText)
							lbl.Color = u.t.Text2
							lbl.MaxLines = 1
							return lbl.Layout(gtx)
						}),
					)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return u.replyX.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.UniformInset(unit.Dp(4)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return icon(gtx, "close", 16, u.t.Text2)
						})
					})
				}),
			)
		})
		call := macro.Stop()
		// latar bar + garis accent kiri
		r := gtx.Dp(6)
		paint.FillShape(gtx.Ops, u.t.SearchBg, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
		paint.FillShape(gtx.Ops, u.t.Accent, clip.Rect{Max: image.Pt(gtx.Dp(3), dims.Size.Y)}.Op())
		call.Add(gtx.Ops)
		return dims
	})
}

// editBanner — bilah "Edit pesan" di atas composer (garis accent + label + teks
// asli + ✕ batal). Sama gaya replyBanner.
func (u *UI) editBanner(gtx layout.Context) layout.Dimensions {
	return layout.Inset{Left: unit.Dp(16), Right: unit.Dp(12), Top: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		macro := op.Record(gtx.Ops)
		dims := layout.Inset{Top: unit.Dp(6), Bottom: unit.Dp(6), Left: unit.Dp(10), Right: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Right: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return icon(gtx, "editpen", 18, u.t.Accent)
					})
				}),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							lbl := material.Label(u.th, 13, "Edit pesan")
							lbl.Color = u.t.Accent
							lbl.Font.Weight = font.Medium
							lbl.MaxLines = 1
							return lbl.Layout(gtx)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							lbl := material.Label(u.th, 13, u.editText)
							lbl.Color = u.t.Text2
							lbl.MaxLines = 1
							return lbl.Layout(gtx)
						}),
					)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return u.editCancel.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.UniformInset(unit.Dp(4)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return icon(gtx, "close", 16, u.t.Text2)
						})
					})
				}),
			)
		})
		call := macro.Stop()
		r := gtx.Dp(6)
		paint.FillShape(gtx.Ops, u.t.SearchBg, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
		paint.FillShape(gtx.Ops, u.t.Accent, clip.Rect{Max: image.Pt(gtx.Dp(3), dims.Size.Y)}.Op())
		call.Add(gtx.Ops)
		return dims
	})
}

// sendOrMic — slot kanan composer: tombol KIRIM (lingkaran accent + ikon kirim,
// klik → sendCurrent) saat ada teks; ikon mikrofon (visual) saat kosong. Cara
// WhatsApp menukar mic↔kirim mengikuti isi.
// recordingBar — bar composer saat merekam voice note: ikon batal (trash) + titik
// merah + timer + label "Merekam…" + tombol kirim (accent). Ketuk batal → buang,
// kirim → finalisasi + SendMedia voice.
func (u *UI) recordingBar(gtx layout.Context) layout.Dimensions {
	h := gtx.Dp(62)
	sz := image.Pt(gtx.Constraints.Max.X, h)
	paint.FillShape(gtx.Ops, u.t.HeadBg, clip.Rect{Max: sz}.Op())
	paint.FillShape(gtx.Ops, u.t.Divider, clip.Rect{Max: image.Pt(sz.X, 1)}.Op())
	for u.recCancel.Clicked(gtx) {
		if u.core != nil {
			u.core.CancelVoiceRecord()
		}
	}
	for u.recSend.Clicked(gtx) {
		if u.core != nil {
			u.core.StopVoiceRecordAndSend(u.selected)
		}
	}
	secs := 0
	if u.core != nil {
		secs = u.core.VoiceRecordSeconds()
	}
	timer := itoa(secs/60) + ":" + pad2(secs%60)
	gtx.Constraints.Min, gtx.Constraints.Max = sz, sz
	return layout.Inset{Left: unit.Dp(12), Right: unit.Dp(10), Top: unit.Dp(11), Bottom: unit.Dp(11)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return u.recCancel.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return icon(gtx, "trash", 22, color.NRGBA{R: 0xe3, G: 0x5d, B: 0x6a, A: 0xff})
					})
				})
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions { // titik merah
				d := gtx.Dp(10)
				paint.FillShape(gtx.Ops, color.NRGBA{R: 0xe3, G: 0x4d, B: 0x4d, A: 0xff}, clip.Ellipse{Max: image.Pt(d, d)}.Op(gtx.Ops))
				return layout.Dimensions{Size: image.Pt(d, d)}
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				l := material.Label(u.th, 15, timer+"  Merekam…")
				l.Color = u.t.Text
				return l.Layout(gtx)
			}),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions { return layout.Dimensions{Size: gtx.Constraints.Min} }),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return u.recSend.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					d := gtx.Dp(40)
					s := image.Pt(d, d)
					paint.FillShape(gtx.Ops, u.t.Accent, clip.Ellipse{Max: s}.Op(gtx.Ops))
					gtx.Constraints.Min, gtx.Constraints.Max = s, s
					return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return icon(gtx, "send", 20, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
					})
				})
			}),
		)
	})
}

// pad2 — angka 0-59 → 2 digit ("03").
func pad2(n int) string {
	if n < 10 {
		return "0" + itoa(n)
	}
	return itoa(n)
}

func (u *UI) sendOrMic(gtx layout.Context) layout.Dimensions {
	for u.sendClick.Clicked(gtx) {
		u.sendCurrent()
	}
	for u.micClick.Clicked(gtx) { // ketuk mic → mulai rekam voice note
		if u.core != nil && u.selected != "" {
			u.core.StartVoiceRecord()
		}
	}
	if strings.TrimSpace(u.editor.Text()) == "" {
		return u.glyphBtn(gtx, &u.micClick, "mic") // kosong → mic (ketuk = rekam)
	}
	return u.sendClick.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		d := gtx.Dp(40)
		sz := image.Pt(d, d)
		paint.FillShape(gtx.Ops, u.t.Accent, clip.Ellipse{Max: sz}.Op(gtx.Ops))
		gtx.Constraints.Min, gtx.Constraints.Max = sz, sz
		return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return icon(gtx, "send", 20, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
		})
	})
}

func (u *UI) glyphBtn(gtx layout.Context, c *widget.Clickable, iconName string) layout.Dimensions {
	return u.glyphBtnTip(gtx, c, iconName, "")
}

// glyphBtnTip — tombol ikon 40px dgn bg-hover bulat + tooltip (di atas) saat hover.
func (u *UI) glyphBtnTip(gtx layout.Context, c *widget.Clickable, iconName, tip string) layout.Dimensions {
	body := func(gtx layout.Context) layout.Dimensions {
		d := gtx.Dp(40)
		sz := image.Pt(d, d)
		gtx.Constraints.Min, gtx.Constraints.Max = sz, sz
		col := u.t.RailIco
		if c != nil && c.Hovered() { // bg bulat + ikon accent saat hover
			rad := d / 2
			paint.FillShape(gtx.Ops, u.t.Hover, clip.RRect{Rect: image.Rectangle{Max: sz}, NW: rad, NE: rad, SE: rad, SW: rad}.Op(gtx.Ops))
			col = u.t.Accent
			if tip != "" {
				m := op.Record(gtx.Ops)
				u.tipAbove(gtx, d, tip)
				op.Defer(gtx.Ops, m.Stop())
			}
		}
		layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return icon(gtx, iconName, 24, col)
		})
		return layout.Dimensions{Size: sz}
	}
	if c == nil {
		return body(gtx)
	}
	return c.Layout(gtx, body)
}

// tipAbove — tooltip kecil DI ATAS tombol (utk composer di bawah layar). Kotak
// inverse-tema + caret bawah menunjuk tombol.
func (u *UI) tipAbove(gtx layout.Context, btnH int, txt string) {
	tipBg, tipFg := u.t.Text, u.t.SidebarBg
	cg := gtx
	cg.Constraints = layout.Constraints{Max: image.Pt(gtx.Dp(240), gtx.Dp(40))}
	m := op.Record(gtx.Ops)
	lbl := material.Label(u.th, 12, txt)
	lbl.Color, lbl.MaxLines = tipFg, 1
	dims := layout.UniformInset(unit.Dp(7)).Layout(cg, lbl.Layout)
	call := m.Stop()
	r := gtx.Dp(6)
	cx := gtx.Dp(20) // ~tengah tombol (40/2)
	y := -dims.Size.Y - gtx.Dp(8)
	off := op.Offset(image.Pt(cx-dims.Size.X/2, y)).Push(gtx.Ops)
	paint.FillShape(gtx.Ops, tipBg, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: r, NE: r, SE: r, SW: r}.Op(gtx.Ops))
	call.Add(gtx.Ops)
	off.Pop()
	// caret segitiga bawah.
	var p clip.Path
	p.Begin(gtx.Ops)
	cw := gtx.Dp(5)
	p.MoveTo(f32.Pt(float32(cx-cw), float32(y+dims.Size.Y)))
	p.LineTo(f32.Pt(float32(cx+cw), float32(y+dims.Size.Y)))
	p.LineTo(f32.Pt(float32(cx), float32(y+dims.Size.Y+cw)))
	p.Close()
	paint.FillShape(gtx.Ops, tipBg, clip.Outline{Path: p.End()}.Op())
}

// sendCurrent — kirim isi composer (teks atau balasan), reset editor + banner +
// indikator mengetik, lalu gulir ke bawah. Dipakai tombol kirim & tombol Enter.
func (u *UI) clearEdit() { u.editTarget, u.editText = "", "" }

// groupMembers — anggota grup terpilih (cache 5s) utk saran @mention.
func (u *UI) groupMembers() []app.GroupMemberDTO {
	if !u.selGroup || u.selected == "" {
		return nil
	}
	if u.core == nil {
		return u.memCache // demo: di-inject via SetMentionDemo
	}
	if u.memCacheJID == u.selected && time.Since(u.memAt) < 5*time.Second {
		return u.memCache
	}
	if gi := u.core.GetGroupInfo(u.selected); gi != nil {
		u.memCache, u.memCacheJID, u.memAt = gi.Participants, u.selected, time.Now()
	}
	return u.memCache
}

// mentionToken — token "@partial" di AKHIR teks (di awal kata). Kembalikan
// (token, indeks '@') atau ("", -1) bila tak sedang mengetik mention.
func mentionToken(text string) (string, int) {
	at := strings.LastIndex(text, "@")
	if at < 0 {
		return "", -1
	}
	if at > 0 {
		if pc := text[at-1]; pc != ' ' && pc != '\n' {
			return "", -1 // '@' di tengah kata (mis. email) → bukan mention
		}
	}
	tok := text[at+1:]
	if strings.ContainsAny(tok, " \n") {
		return "", -1 // sudah ada spasi → mention selesai
	}
	return tok, at
}

// mentionMatches — anggota grup yg cocok token (nama/nomor, maks 6).
func (u *UI) mentionMatches(token string) []app.GroupMemberDTO {
	q := strings.ToLower(token)
	out := make([]app.GroupMemberDTO, 0, 6)
	for _, m := range u.groupMembers() {
		if u.selfJID != "" && jidUser(m.JID) == jidUser(u.selfJID) {
			continue // jangan sarankan diri sendiri
		}
		nm := m.Name
		if nm == "" {
			nm = jidUser(m.JID)
		}
		if q == "" || strings.Contains(strings.ToLower(nm), q) || strings.Contains(jidUser(m.JID), q) {
			out = append(out, m)
			if len(out) >= 6 {
				break
			}
		}
	}
	return out
}

// mentionSuggest — popup saran @mention di atas composer (grup, saat mengetik @).
func (u *UI) mentionSuggest(gtx layout.Context) layout.Dimensions {
	if !u.selGroup {
		return layout.Dimensions{}
	}
	tok, at := mentionToken(u.editor.Text())
	if at < 0 {
		return layout.Dimensions{}
	}
	matches := u.mentionMatches(tok)
	if len(matches) == 0 {
		return layout.Dimensions{}
	}
	if len(u.mentionClicks) < len(matches) {
		u.mentionClicks = make([]widget.Clickable, len(matches))
	}
	for i := range matches { // klik → sisip "@<nomor> " + catat jid
		if i >= len(u.mentionClicks) {
			break
		}
		if u.mentionClicks[i].Clicked(gtx) {
			m := matches[i]
			num := jidUser(m.JID)
			text := u.editor.Text()
			u.editor.SetText(text[:at] + "@" + num + " ")
			u.editor.SetCaret(len([]rune(u.editor.Text())), len([]rune(u.editor.Text())))
			u.composeMentions[num] = m.JID // catat jid utk SendTextMentioned
		}
	}
	return layout.Inset{Left: unit.Dp(8), Right: unit.Dp(8), Bottom: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		macro := op.Record(gtx.Ops)
		children := make([]layout.FlexChild, 0, len(matches))
		for i := range matches {
			m, idx := matches[i], i
			children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return u.mentionClicks[idx].Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					nm := m.Name
					if nm == "" {
						nm = jidUser(m.JID)
					}
					return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						gtx.Constraints.Min.X = gtx.Constraints.Max.X
						return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions { return u.avatar(gtx, nm, m.JID, 32) }),
							layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								l := material.Label(u.th, 14.5, nm)
								l.Color, l.MaxLines = u.t.Text, 1
								return l.Layout(gtx)
							}),
						)
					})
				})
			}))
		}
		dims := layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
		call := macro.Stop()
		rr := gtx.Dp(12)
		paint.FillShape(gtx.Ops, u.t.Bg2, clip.RRect{Rect: image.Rectangle{Max: dims.Size}, NW: rr, NE: rr, SE: rr, SW: rr}.Op(gtx.Ops))
		call.Add(gtx.Ops)
		return dims
	})
}

// collectMentions — jid mention yg token "@<nomor>"-nya masih ada di teks.
func (u *UI) collectMentions(text string) []string {
	if !u.selGroup || len(u.composeMentions) == 0 {
		return nil
	}
	out := make([]string, 0, len(u.composeMentions))
	for num, jid := range u.composeMentions {
		if strings.Contains(text, "@"+num) {
			out = append(out, jid)
		}
	}
	return out
}

func (u *UI) sendCurrent() {
	txt := strings.TrimSpace(u.editor.Text())
	if u.editTarget != "" { // mode edit → ubah pesan terkirim (SendEdit)
		if txt != "" && u.core != nil && u.selected != "" {
			u.core.EditMessage(u.selected, u.editTarget, txt)
			u.messages = u.core.GetMessages(u.selected)
		}
		u.clearEdit()
		u.editor.SetText("")
		return
	}
	if txt != "" && u.core != nil && u.selected != "" {
		mentions := u.collectMentions(txt)
		switch {
		case u.replyTo != "": // mode balas → kutip pesan
			u.core.Reply(u.selected, txt, u.replyTo, u.replyName, u.replyText)
		case len(mentions) > 0: // ada @mention grup → kirim dgn daftar jid
			u.core.SendTextMentioned(u.selected, txt, mentions)
		default:
			u.core.SendText(u.selected, txt)
		}
		u.messages = u.core.GetMessages(u.selected)
	}
	for k := range u.composeMentions { // reset mention setelah kirim
		delete(u.composeMentions, k)
	}
	if u.core != nil && u.selected != "" && u.typingSent {
		u.core.SendTyping(u.selected, false, false) // berhenti mengetik
		u.typingSent = false
	}
	u.editor.SetText("")
	delete(u.drafts, u.selected) // terkirim → buang draft chat ini
	u.clearReply()
	u.msgList.ScrollTo(len(u.messages)) // setelah kirim → gulir ke bawah
}

func (u *UI) composerPill(gtx layout.Context) layout.Dimensions {
	rr := gtx.Dp(22)
	minH := gtx.Dp(40)
	maxH := gtx.Dp(140) // ~6 baris lalu menggulir di dalam (ala WhatsApp)
	// Enter → kirim (SubmitEvent); Shift+Enter → baris baru (editor SingleLine=false,
	// ditangani native widget.Editor). core nil (demo) → tak kirim.
	for {
		ev, ok := u.editor.Update(gtx)
		if !ok {
			break
		}
		switch ev.(type) {
		case widget.ChangeEvent:
			// indikator "mengetik" keluar: kirim composing sekali saat mulai isi,
			// stop saat kosong (throttle via typingSent agar tak spam socket).
			if u.core != nil && u.selected != "" {
				typing := strings.TrimSpace(u.editor.Text()) != ""
				if typing != u.typingSent {
					u.core.SendTyping(u.selected, typing, false)
					u.typingSent = typing
				}
			}
		case widget.SubmitEvent:
			u.sendCurrent() // Enter → kirim
		}
	}
	// ukur editor (membungkus + menggulir dalam maxH), lalu gambar pil setinggi itu.
	egtx := gtx
	egtx.Constraints.Min = image.Point{}
	if egtx.Constraints.Max.Y > maxH {
		egtx.Constraints.Max.Y = maxH
	}
	macro := op.Record(gtx.Ops)
	edDims := layout.Inset{Left: unit.Dp(16), Right: unit.Dp(16), Top: unit.Dp(9), Bottom: unit.Dp(9)}.Layout(egtx, func(gtx layout.Context) layout.Dimensions {
		ed := material.Editor(u.th, &u.editor, "Ketik pesan")
		ed.Color = u.t.Text
		ed.HintColor = u.t.Text2
		ed.TextSize = 15
		return ed.Layout(gtx)
	})
	call := macro.Stop()
	h := edDims.Size.Y
	if h < minH {
		h = minH
	}
	psz := image.Pt(gtx.Constraints.Max.X, h)
	paint.FillShape(gtx.Ops, u.t.SearchBg, clip.RRect{Rect: image.Rectangle{Max: psz}, NW: rr, NE: rr, SE: rr, SW: rr}.Op(gtx.Ops))
	call.Add(gtx.Ops)
	return layout.Dimensions{Size: psz}
}

// ---- helpers ----
func initial(name string) string {
	name = strings.TrimSpace(name)
	for _, r := range name {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return strings.ToUpper(string(r))
		}
	}
	return "?"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [12]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
