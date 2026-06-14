<script>
  import Bubble from "./Bubble.svelte";
  import { t } from "../i18n.js";
  import { tick, beforeUpdate, afterUpdate } from "svelte";
  import { loadOlder, jumpMsg, wallpapers, lightbox } from "../../stores.js";
  export let messages = [];
  export let group = false;
  export let chatId;
  export let peerName = "";
  export let firstUnreadId = null; // id pesan pertama belum dibaca (pembatas)
  export let unreadCount = 0;      // jumlah belum dibaca (label pembatas)

  let box;            // wadah scroll
  let atBottom = true;
  let loadingOlder = false, noMore = false;
  let newCount = 0;          // pesan baru selagi scroll ke atas (badge FAB)
  let floatDate = "";        // tanggal mengambang saat scroll
  let floatVisible = false;
  let _floatTimer;
  let _lastMsgLen = 0;
  $: chatId, (noMore = false, newCount = 0, _scrolledUnread = null, _lastMsgLen = 0); // reset saat ganti chat
  // Pesan bertambah (mis. history on-demand tiba) → buka lagi kemungkinan load-older.
  $: if (messages && messages.length > _lastMsgLen) { noMore = false; _lastMsgLen = messages.length; }

  // Gulir ke pembatas "belum dibaca" begitu ia muncul (dihitung sedikit setelah
  // pesan termuat) — sekali per chat.
  let _scrolledUnread = null;
  $: if (firstUnreadId && firstUnreadId !== _scrolledUnread && box && items) {
    _scrolledUnread = firstUnreadId;
    tick().then(() => { const d = box && box.querySelector(".unread-divider"); if (d) d.scrollIntoView({ block: "center" }); });
  }

  // === Scroll behavior (pola kanonik beforeUpdate/afterUpdate) ===
  // follow=true → "menempel" ke bawah (ikut pesan baru). User scroll-up → false.
  let follow = true;
  let snapNearBottom = true;     // snapshot SEBELUM DOM update
  let prevChat = null, prevLastId = null, prevLen = 0;

  function nearBottom() {
    return box ? box.scrollHeight - box.scrollTop - box.clientHeight < 80 : true;
  }
  function pin() { if (box) box.scrollTop = box.scrollHeight; }

  function onScroll() {
    if (!box) return;
    follow = nearBottom();            // user di bawah → ikut; scroll-up → lepas
    atBottom = follow;
    if (follow) newCount = 0;
    if (box.scrollTop < 120) maybeLoadOlder();
    updateFloatDate();
  }

  beforeUpdate(() => { snapNearBottom = nearBottom(); });

  afterUpdate(() => {
    if (!box) return;
    const last = messages.length ? messages[messages.length - 1] : null;
    const lastId = last ? last.id || "" : "";
    const lastMine = last ? last.dir === "out" : false;

    if (chatId !== prevChat) {            // (1) buka chat → ke bawah
      prevChat = chatId; prevLastId = lastId; prevLen = messages.length;
      follow = true; atBottom = true; newCount = 0;
      pin(); requestAnimationFrame(pin); setTimeout(pin, 120); setTimeout(pin, 350);
      return;
    }
    const isNew = lastId !== prevLastId;  // pesan BARU di ujung (bukan reaksi/edit/prepend)
    const added = Math.max(0, messages.length - prevLen);
    prevLastId = lastId; prevLen = messages.length;
    if (!isNew) return;                   // (7) reaksi/edit/receipt → diam

    if (lastMine) {                       // (4) kirim sendiri → SELALU ke bawah
      follow = true; atBottom = true; newCount = 0; pin();
    } else if (snapNearBottom) {          // (2) di bawah + pesan baru → ikut
      atBottom = true; newCount = 0; pin();
    } else {                              // (3) scroll-up + pesan baru → diam + badge
      newCount += added || 1;
    }
  });
  // Tanggal pesan teratas yang terlihat → pil mengambang (Telegram-style).
  function updateFloatDate() {
    if (!box) return;
    const top = box.getBoundingClientRect().top;
    const bubbles = box.querySelectorAll("[data-ts]");
    for (const el of bubbles) {
      if (el.getBoundingClientRect().bottom > top + 8) {
        const ts = +el.getAttribute("data-ts");
        if (ts) floatDate = dayLabel(ts);
        break;
      }
    }
    floatVisible = true;
    clearTimeout(_floatTimer);
    _floatTimer = setTimeout(() => (floatVisible = false), 1400);
  }
  // Scroll ke atas → muat riwayat lebih lama, jaga posisi (tak melompat).
  async function maybeLoadOlder() {
    if (loadingOlder || noMore || !box) return;
    loadingOlder = true;
    const prevH = box.scrollHeight, prevT = box.scrollTop;
    const n = await loadOlder(chatId);
    if (n === 0) noMore = true;
    await tick();
    box.scrollTop = box.scrollHeight - prevH + prevT;
    loadingOlder = false;
  }
  function toBottom(smooth = true) {
    if (!box) return;
    box.scrollTo({ top: box.scrollHeight, behavior: smooth ? "smooth" : "auto" });
  }

  // Buang pesan teks kosong (tak ada isi & bukan media) → tak ada bubble kosong.
  function empty(m) {
    if (m.type === "deleted") return false; // pesan dihapus → tetap tampil (placeholder)
    const media = m.type === "image" || m.type === "video" || m.type === "sticker" || m.type === "voice";
    return !media && m.type !== "day" && m.type !== "system" && m.type !== "unread" && !(m.text && m.text.trim()) && !m.thumb;
  }

  $: items = build(messages, group, firstUnreadId, unreadCount);
  function dayLabel(ts) {
    if (!ts) return "";
    const d = new Date(ts * 1000), now = new Date();
    const day = (x) => new Date(x.getFullYear(), x.getMonth(), x.getDate()).getTime();
    const diff = Math.round((day(now) - day(d)) / 86400000);
    if (diff <= 0) return $t("today");
    if (diff === 1) return $t("yesterday");
    return d.toLocaleDateString(undefined, { day: "numeric", month: "long", year: d.getFullYear() === now.getFullYear() ? undefined : "numeric" });
  }

  // Buka album di lightbox (swipe antar item).
  function openAlbum(albumItems, start) {
    const items = albumItems.map((a) => ({ url: `/media/${chatId}/${a.id}`, type: a.mtype, caption: "" }));
    lightbox.set({ items, i: start });
  }

  function build(msgs, grp, unreadId, unreadN) {
    let prevKey = null;
    let prevDay = null;
    const out = [];
    msgs.forEach((m, idx) => {
      if (empty(m)) return; // idx tetap index asli (untuk hapus/reaksi)
      // Pembatas "belum dibaca" tepat sebelum pesan pertama yg belum dibaca.
      if (unreadId && m.id === unreadId) {
        out.push({ m: { type: "unread", count: unreadN }, idx: "unread", firstOfRun: false });
        prevKey = null; // putus runtun
      }
      // Sisipkan pemisah hari saat tanggal berganti (abaikan chip/sistem bawaan).
      if (m.type !== "day" && m.type !== "system" && m.type !== "unread" && m.ts) {
        const d = new Date(m.ts * 1000);
        const key = `${d.getFullYear()}-${d.getMonth()}-${d.getDate()}`;
        if (key !== prevDay) {
          prevDay = key;
          out.push({ m: { type: "day", label: dayLabel(m.ts) }, idx: "day-" + idx, firstOfRun: false });
          prevKey = null;
        }
      }
      const isBubble = m.type === "text" || m.type === "image" || m.type === "video" || m.type === "sticker" || m.type === "voice";
      let firstOfRun = true;
      if (isBubble) {
        // Kelompokkan pesan beruntun: per arah, + per pengirim utk grup-masuk.
        const key = m.dir + "|" + (grp && m.dir === "in" ? m.sender || m.senderId || "" : "");
        firstOfRun = key !== prevKey;
        prevKey = key;
      } else {
        prevKey = null; // chip hari/sistem memutus runtun
      }
      out.push({ m, idx, firstOfRun });
    });
    return groupAlbums(out);
  }

  // Gabung 2+ foto/video beruntun (tanpa caption, arah/pengirim sama, tak
  // dipisah chip/hari) → satu item "album" → grid (ala WhatsApp).
  function groupAlbums(arr) {
    const isMedia = (it) => it.m && (it.m.type === "image" || it.m.type === "video") && !(it.m.caption || "").trim() && !it.m.quote && !(it.m.reactions && it.m.reactions.length);
    const res = [];
    let i = 0;
    while (i < arr.length) {
      if (isMedia(arr[i])) {
        let j = i + 1;
        while (j < arr.length && isMedia(arr[j]) && arr[j].m.dir === arr[i].m.dir && (arr[j].m.sender || "") === (arr[i].m.sender || "")) j++;
        if (j - i >= 2) {
          const group = arr.slice(i, j);
          res.push({ m: { type: "album", items: group.map((g) => ({ id: g.m.id, idx: g.idx, mtype: g.m.type, time: g.m.time })), dir: arr[i].m.dir, time: group[group.length - 1].m.time }, idx: "album-" + arr[i].idx, firstOfRun: arr[i].firstOfRun });
          i = j;
          continue;
        }
      }
      res.push(arr[i]);
      i++;
    }
    return res;
  }

  // Lompat + highlight ke pesan (dari hasil pencarian). Best-effort: hanya bila
  // pesannya termuat (dalam ~200 terakhir); kalau tidak, biarkan di posisi bawah.
  $: if ($jumpMsg && box && items) tick().then(() => flashTo($jumpMsg));
  function flashTo(mid) {
    const el = box && box.querySelector(`[data-mid="${(window.CSS && CSS.escape) ? CSS.escape(mid) : mid}"]`);
    if (el) {
      el.scrollIntoView({ block: "center" });
      el.classList.add("flash");
      setTimeout(() => el.classList.remove("flash"), 1500);
    }
    jumpMsg.set(null);
  }

  // Lompat ke tanggal: pilih tanggal → ke pesan pertama hari itu (dalam yang
  // termuat; muat lebih lama beberapa kali bila perlu).
  let dateInput;
  function openDatePick() {
    if (!dateInput) return;
    if (dateInput.showPicker) dateInput.showPicker(); else dateInput.click();
  }
  async function onPickDate(e) {
    const v = e.target.value;
    if (!v) return;
    const target = new Date(v + "T00:00:00").getTime() / 1000;
    for (let tries = 0; tries < 10; tries++) {
      const m = messages.find((x) => x.ts && x.ts >= target);
      if (m) { jumpMsg.set(m.id); return; }
      if (noMore || !box || box.scrollTop > 120) break;
      await maybeLoadOlder();
      await tick();
    }
    // tak ketemu (lebih lama dari yg termuat) → ke paling atas yg ada.
    if (messages.length) jumpMsg.set(messages[0].id);
  }
</script>

<div class="msg-wrap">
  {#if floatDate}
    <button class="float-date {floatVisible ? 'on' : ''}" on:click={openDatePick} title={$t("jump_to_date")}><span>{floatDate}</span></button>
    <input type="date" bind:this={dateInput} on:change={onPickDate} style="position:absolute;width:0;height:0;opacity:0;pointer-events:none" />
  {/if}
  <div class="messages" bind:this={box} on:scroll={onScroll} style={$wallpapers[chatId] ? `background-color:${$wallpapers[chatId]}` : ""}>
    {#each items as it (it.m.id || it.idx)}
      {#if it.m.type === "day"}
        <div class="day-chip"><span>{it.m.label || $t("today")}</span></div>
      {:else if it.m.type === "system"}
        <div class="system-msg">
          <svg class="lock" viewBox="0 0 24 24"><rect x="5" y="11" width="14" height="9" rx="2"/><path d="M8 11V8a4 4 0 0 1 8 0v3"/></svg>
          {$t("enc_notice")}
        </div>
      {:else if it.m.type === "unread"}
        <div class="unread-divider"><span>{$t("unread_count", { n: it.m.count })}</span></div>
      {:else if it.m.type === "album"}
        <div class="msg {it.m.dir}">
          <div class="album-grid n{Math.min(it.m.items.length, 4)} {it.m.dir}">
            {#each it.m.items.slice(0, 4) as a, ai}
              <button class="album-cell" on:click={() => openAlbum(it.m.items, ai)}>
                <img src={`/media/${chatId}/${a.id}`} alt="" loading="lazy" on:error={(e) => (e.target.style.visibility = 'hidden')} />
                {#if a.mtype === "video"}<span class="album-play">▶</span>{/if}
                {#if ai === 3 && it.m.items.length > 4}<span class="album-more">+{it.m.items.length - 4}</span>{/if}
              </button>
            {/each}
            <span class="album-time">{it.m.time}</span>
          </div>
        </div>
      {:else}
        <Bubble msg={it.m} {group} {chatId} {peerName} idx={it.idx} firstOfRun={it.firstOfRun} />
      {/if}
    {/each}
  </div>
  {#if !atBottom}
    <button class="scroll-fab" on:click={() => { follow = true; atBottom = true; newCount = 0; toBottom(); }} aria-label={$t("scroll_bottom")}>
      {#if newCount > 0}<span class="fab-badge">{newCount > 99 ? "99+" : newCount}</span>{/if}
      <svg viewBox="0 0 24 24"><path d="M6 9l6 6 6-6"/></svg>
    </button>
  {/if}
</div>
