<script>
  import { chats, forwardDraft, forwardMessage } from "../../stores.js";
  import { t } from "../i18n.js";
  import Avatar from "../common/Avatar.svelte";

  let q = "";
  $: list = $chats.filter((c) => (c.name || "").toLowerCase().includes(q.toLowerCase()));
  function pick(c) {
    const d = $forwardDraft;
    if (d) {
      const idxs = d.idxs || (d.idx != null ? [d.idx] : []);
      for (const idx of idxs) forwardMessage(d.chat, idx, c.id);
    }
    forwardDraft.set(null);
    q = "";
  }
  function close() { forwardDraft.set(null); q = ""; }
</script>

{#if $forwardDraft}
  <button class="modal-backdrop" aria-label={$t("close")} on:click={close}></button>
  <div class="fwd-modal" role="dialog">
    <div class="fwd-head">{$t("forward_action")}</div>
    <input class="fwd-search" placeholder={$t("search")} bind:value={q} />
    <div class="fwd-list">
      {#each list as c (c.id)}
        <button class="fwd-row" on:click={() => pick(c)}>
          <Avatar name={c.name} color={c.color} photo={c.photo} group={c.group} sm={true} />
          <span class="fwd-name">{c.name}</span>
        </button>
      {/each}
    </div>
  </div>
{/if}
