<script>
  import { t } from "./i18n.js";
  import { qrImage } from "../stores.js";
</script>

<div class="login">
  <div class="login-bar">WhatsApp Lite</div>
  <div class="login-card">
    <div class="login-left">
      <h2>{$t("login_title")}</h2>
      <ol class="login-steps">
        <li>{$t("login_s1")}</li>
        <li>{$t("login_s2")}</li>
        <li>{$t("login_s3")}</li>
      </ol>
    </div>
    <div class="login-right">
      {#if $qrImage}
        <img class="qr-img" src={$qrImage} alt="QR" />
      {:else}
      <svg class="qr" viewBox="0 0 96 96">
        <!-- finder patterns -->
        <rect x="4" y="4" width="22" height="22"/><rect class="w" x="8" y="8" width="14" height="14"/><rect x="11" y="11" width="8" height="8"/>
        <rect x="70" y="4" width="22" height="22"/><rect class="w" x="74" y="8" width="14" height="14"/><rect x="77" y="11" width="8" height="8"/>
        <rect x="4" y="70" width="22" height="22"/><rect class="w" x="8" y="74" width="14" height="14"/><rect x="11" y="77" width="8" height="8"/>
        <!-- data modules -->
        <rect x="34" y="6" width="5" height="5"/><rect x="46" y="6" width="5" height="5"/><rect x="58" y="10" width="5" height="5"/>
        <rect x="34" y="18" width="5" height="5"/><rect x="52" y="18" width="5" height="5"/><rect x="64" y="18" width="5" height="5"/>
        <rect x="6" y="34" width="5" height="5"/><rect x="18" y="34" width="5" height="5"/><rect x="40" y="34" width="5" height="5"/><rect x="58" y="34" width="5" height="5"/><rect x="76" y="34" width="5" height="5"/><rect x="88" y="34" width="5" height="5"/>
        <rect x="30" y="42" width="5" height="5"/><rect x="48" y="42" width="5" height="5"/><rect x="66" y="42" width="5" height="5"/>
        <rect x="10" y="48" width="5" height="5"/><rect x="38" y="50" width="5" height="5"/><rect x="56" y="50" width="5" height="5"/><rect x="84" y="50" width="5" height="5"/>
        <rect x="34" y="60" width="5" height="5"/><rect x="52" y="62" width="5" height="5"/><rect x="70" y="60" width="5" height="5"/><rect x="88" y="60" width="5" height="5"/>
        <rect x="40" y="72" width="5" height="5"/><rect x="58" y="74" width="5" height="5"/><rect x="76" y="72" width="5" height="5"/>
        <rect x="34" y="84" width="5" height="5"/><rect x="50" y="86" width="5" height="5"/><rect x="66" y="84" width="5" height="5"/><rect x="84" y="86" width="5" height="5"/>
      </svg>
      {/if}
      <div class="login-waiting">{$t("login_waiting")}</div>
    </div>
  </div>
  <div class="login-hint">{$t("login_hint")}</div>
</div>
