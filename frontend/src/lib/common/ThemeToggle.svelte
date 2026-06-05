<script>
  import { theme } from "../../stores.js";
  function toggle() {
    theme.update((t) => (t === "dark" ? "light" : "dark"));
  }
</script>

<button class="theme-toggle" on:click={toggle} title="Ganti tema">
  {$theme === "dark" ? "☀️" : "🌙"}
</button>
