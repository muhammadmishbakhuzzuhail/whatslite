<script>
  import { t } from "../i18n.js";
  export let titleKey = "";
</script>

<header class="pane-head"><h2>{$t(titleKey)}</h2></header>
<div class="placeholder-pane">
  <div>
    <div class="ph-ico">
      <svg viewBox="0 0 24 24"><circle cx="12" cy="12" r="9"/><path d="M12 8v4l3 2"/></svg>
    </div>
    <div class="ph-title">{$t(titleKey)}</div>
    <div>{$t("coming_soon")}</div>
  </div>
</div>
