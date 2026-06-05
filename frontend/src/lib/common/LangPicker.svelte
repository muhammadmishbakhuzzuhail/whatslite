<script>
  // Dropdown bahasa kustom (ganti <select> native yg jelek + tak bisa di-style).
  // Popup ber-posisi FIXED (lolos dari overflow scroll induk) + kotak cari.
  // options: [{code, name, en?}]  (name = nama tampil; en = nama Inggris utk cari)
  import { t } from "../i18n.js";
  export let options = [];
  export let value = "";
  export let onSelect = () => {};

  let open = false, q = "", btn;
  let popTop = 0, popLeft = 0, popW = 250, popMax = 320;

  $: current = options.find((o) => o.code === value);
  $: filtered = (() => {
    const s = q.trim().toLowerCase();
    if (!s) return options;
    return options.filter((o) =>
      (o.name || "").toLowerCase().includes(s) ||
      (o.en || "").toLowerCase().includes(s) ||
      (o.code || "").toLowerCase().includes(s));
  })();

  function toggle() {
    if (!open && btn) {
      const r = btn.getBoundingClientRect();
      popW = Math.max(250, r.width);
      popLeft = Math.min(r.left, window.innerWidth - popW - 12);
      popTop = r.bottom + 6;
      popMax = Math.max(180, window.innerHeight - popTop - 16);
    }
    open = !open; q = "";
  }
  function pick(code) { onSelect(code); open = false; q = ""; }
</script>

<div class="lp">
  <button class="lp-btn" bind:this={btn} on:click|stopPropagation={toggle} aria-haspopup="listbox" aria-expanded={open}>
    <span class="lp-cur">{current ? current.name : value}</span>
    <svg class="lp-chev {open ? 'up' : ''}" viewBox="0 0 12 12"><path d="M2.5 4.5L6 8l3.5-3.5" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg>
  </button>
  {#if open}
    <button class="lp-back" aria-label={$t("close")} on:click={() => (open = false)}></button>
    <div class="lp-pop" style="top:{popTop}px; left:{popLeft}px; width:{popW}px; max-height:{popMax}px">
      <input class="lp-search" placeholder={$t("search")} bind:value={q} autofocus on:click|stopPropagation />
      <div class="lp-list">
        {#each filtered as o (o.code)}
          <button class="lp-opt {o.code === value ? 'on' : ''}" on:click={() => pick(o.code)}>
            <span class="lp-name">{o.name}</span>
            {#if o.en && o.en !== o.name}<span class="lp-sub">{o.en}</span>{/if}
            {#if o.code === value}<span class="lp-check">✓</span>{/if}
          </button>
        {/each}
        {#if filtered.length === 0}<div class="lp-empty">—</div>{/if}
      </div>
    </div>
  {/if}
</div>

<style>
  .lp { position: relative; }
  .lp-btn { display: inline-flex; align-items: center; gap: 8px; max-width: 220px;
    padding: 7px 11px; border: 1px solid var(--divider); border-radius: 9px;
    background: var(--search-bg); color: var(--text); font: inherit; font-size: 13px;
    cursor: pointer; transition: border-color .12s; }
  .lp-btn:hover { border-color: var(--accent); }
  .lp-cur { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .lp-chev { width: 12px; height: 12px; flex: 0 0 auto; color: var(--text2); transition: transform .15s; }
  .lp-chev.up { transform: rotate(180deg); }
  .lp-back { position: fixed; inset: 0; z-index: 80; background: transparent; border: 0; cursor: default; }
  .lp-pop { position: fixed; z-index: 81; display: flex; flex-direction: column;
    background: var(--sidebar-bg, var(--bg)); border: 1px solid var(--divider);
    border-radius: 12px; box-shadow: var(--shadow-lg); overflow: hidden; }
  .lp-search { margin: 8px; padding: 9px 12px; border: 1px solid var(--divider); border-radius: 9px;
    background: var(--search-bg); color: var(--text); font: inherit; font-size: 13px; outline: none; }
  .lp-search:focus { border-color: var(--accent); }
  .lp-list { overflow-y: auto; padding: 0 6px 8px; }
  .lp-opt { display: flex; align-items: center; gap: 8px; width: 100%; text-align: left;
    padding: 9px 10px; border: 0; border-radius: 8px; background: none; color: var(--text);
    font: inherit; font-size: 14px; cursor: pointer; }
  .lp-opt:hover { background: var(--hover); }
  .lp-opt.on { background: color-mix(in srgb, var(--accent) 14%, transparent); }
  .lp-name { flex: 0 0 auto; }
  .lp-sub { flex: 1; min-width: 0; font-size: 11.5px; color: var(--text2); overflow: hidden;
    text-overflow: ellipsis; white-space: nowrap; }
  .lp-check { flex: 0 0 auto; color: var(--accent); font-weight: 700; }
  .lp-empty { padding: 16px; text-align: center; color: var(--text2); }
</style>
