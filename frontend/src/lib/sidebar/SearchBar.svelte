<script>
  import { search } from "../../stores.js";
  import { t } from "../i18n.js";
</script>

<div class="search-wrap">
  <div class="search">
    <svg class="search-ico" viewBox="0 0 24 24"><circle cx="11" cy="11" r="7"/><path d="M21 21l-4.3-4.3"/></svg>
    <input type="text" placeholder={$t("search_placeholder")} bind:value={$search} />
  </div>
</div>
