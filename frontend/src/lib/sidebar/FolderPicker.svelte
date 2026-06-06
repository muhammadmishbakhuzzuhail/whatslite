<script>
  // Pilih folder untuk satu chat (checkbox) + buat folder baru.
  import { folders, folderPickFor, toggleChatFolder, addFolder, deleteFolder, askPrompt, chats } from "../../stores.js";
  import { t } from "../i18n.js";
  $: jid = $folderPickFor;
  $: chat = jid ? $chats.find((c) => c.id === jid) : null;
  function close() { folderPickFor.set(null); }
  function newFolder() { askPrompt($t("folder_new"), "", (name) => addFolder(name)); }
</script>

{#if jid}
  <button class="modal-backdrop" aria-label={$t("close")} on:click={close}></button>
  <div class="nc-modal" role="dialog">
    <div class="nc-head">
      <span>{$t("folders")}{chat ? ` · ${chat.name}` : ""}</span>
      <button class="icon-btn" on:click={close} aria-label={$t("close")}><svg viewBox="0 0 24 24"><path d="M6 6l12 12M18 6L6 18"/></svg></button>
    </div>
    <div class="fp-list">
      {#each $folders as f (f.name)}
        <label class="fp-row">
          <input type="checkbox" checked={f.jids.includes(jid)} on:change={() => toggleChatFolder(f.name, jid)} />
          <span class="fp-name">{f.name}</span>
          <button class="fp-del" title={$t("delete")} on:click|preventDefault={() => deleteFolder(f.name)}>✕</button>
        </label>
      {/each}
      {#if $folders.length === 0}<div class="fp-empty">{$t("no_folders")}</div>{/if}
    </div>
    <button class="nc-create" on:click={newFolder}>+ {$t("folder_new")}</button>
  </div>
{/if}

<style>
  .fp-list { max-height: 50vh; overflow-y: auto; margin-bottom: 10px; }
  .fp-row { display: flex; align-items: center; gap: 10px; padding: 10px 8px; border-radius: 8px; cursor: pointer; }
  .fp-row:hover { background: var(--bg2); }
  .fp-name { flex: 1; color: var(--text); }
  .fp-del { background: none; border: 0; color: var(--text2); cursor: pointer; }
  .fp-empty { text-align: center; color: var(--text2); padding: 16px; font-size: 13px; }
</style>
