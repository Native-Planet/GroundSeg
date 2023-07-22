<script>
  import { limbo } from '$lib/stores/display'
  import { connected } from '$lib/stores/websocket'
</script>
<div class="cover" style="display:{$limbo ? "block" : "none"};">
</div>

<style>
  .cover {
    position: absolute;
    top: 0;
    left: 0;
    width: 100vw;
    height: 100vh;
    background: var(--btn-secondary);
  }
</style>
