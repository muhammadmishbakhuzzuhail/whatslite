<script>
  import { lockState, setPin, tryUnlock } from "../stores.js";
  import { t } from "./i18n.js";

  let pin = "";
  let first = "";
  let stage = "first"; // untuk setup
  let err = false;
  $: setup = $lockState === "setup";

  function add(d) {
    if (pin.length >= 4) return;
    err = false;
    pin += d;
    if (pin.length === 4) setTimeout(done, 130);
  }
  function back() { pin = pin.slice(0, -1); }
  function done() {
    if (setup) {
      if (stage === "first") { first = pin; pin = ""; stage = "confirm"; }
      else if (pin === first) { setPin(pin); }
      else { err = true; pin = ""; first = ""; stage = "first"; }
    } else if (!tryUnlock(pin)) {
      err = true; pin = "";
    }
  }

  $: title = setup
    ? stage === "first" ? $t("lock_set") : $t("lock_confirm_pin")
    : $t("lock_enter");
</script>

<div class="lock-screen">
  <div class="lock-logo">
    <svg viewBox="0 0 24 24"><rect x="5" y="11" width="14" height="9" rx="2"/><path d="M8 11V8a4 4 0 0 1 8 0v3"/></svg>
  </div>
  <div class="lock-title">{title}</div>
  <div class="pin-dots">
    {#each [0, 1, 2, 3] as i}<span class="dot {pin.length > i ? 'on' : ''}"></span>{/each}
  </div>
  {#if err}<div class="lock-err">{$t("lock_wrong")}</div>{/if}
  <div class="keypad">
    {#each [1, 2, 3, 4, 5, 6, 7, 8, 9] as n}<button on:click={() => add(String(n))}>{n}</button>{/each}
    <span></span>
    <button on:click={() => add("0")}>0</button>
    <button class="kp-back" on:click={back} aria-label="Backspace">⌫</button>
  </div>
</div>
