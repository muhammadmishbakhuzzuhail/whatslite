<script>
  // status: sending | sent | delivered | read
  export let status = "sent";
</script>

{#if status === "sending"}
  <svg class="ticks clock" viewBox="0 0 16 16"><circle cx="8" cy="8" r="6" /><path d="M8 4.7V8l2.2 1.3" /></svg>
{:else if status === "sent"}
  <svg class="ticks" viewBox="0 0 18 14"><path d="M3 7.5l3.5 3.5L14 4" /></svg>
{:else}
  <svg class="ticks {status === 'read' ? 'read' : ''}" viewBox="0 0 18 14">
    <path d="M1 7.5l3.2 3.2L10 4" />
    <path d="M7 10.7L12.8 4" />
  </svg>
{/if}
