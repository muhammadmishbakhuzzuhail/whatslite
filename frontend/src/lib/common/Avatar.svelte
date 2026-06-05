<script>
  import { initial } from "../util.js";
  export let name = "";
  export let color = "#6a9e3d";
  export let photo = "";
  export let group = false;
  export let sm = false;
  export let tiny = false;
  let imgErr = false;
  $: photo, (imgErr = false); // reset bila sumber berubah
</script>

{#if photo && !imgErr}
  <img class="avatar photo {sm ? 'sm' : ''} {tiny ? 'tiny' : ''}" src={photo} alt={name}
    loading="lazy" decoding="async" on:error={() => (imgErr = true)} />
{:else if group}
  <div class="avatar group {sm ? 'sm' : ''} {tiny ? 'tiny' : ''}" style="--c:{color}">
    <svg viewBox="0 0 24 24"><path d="M16 11c1.66 0 3-1.34 3-3s-1.34-3-3-3-3 1.34-3 3 1.34 3 3 3zm-8 0c1.66 0 3-1.34 3-3S9.66 5 8 5 5 6.34 5 8s1.34 3 3 3zm0 2c-2.33 0-7 1.17-7 3.5V18h14v-1.5c0-2.33-4.67-3.5-7-3.5zm8 0c-.29 0-.62.02-.97.05 1.16.84 1.97 1.99 1.97 3.45V18h6v-1.5c0-2.33-4.67-3.5-7-3.5z"/></svg>
  </div>
{:else}
  <div class="avatar {sm ? 'sm' : ''} {tiny ? 'tiny' : ''}" style="--c:{color}"><span>{initial(name)}</span></div>
{/if}
