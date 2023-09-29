<script>
  import { connected } from '$lib/stores/websocket';
</script>
{#if !$connected}
  <div class="text">CONNECTING</div>
{/if}
<style>
  .text {
    position: fixed;
    top: 20px;
    right: 20px;
    font-family: var(--title-font);
    color: var(--text-card-color);
    font-size: 20px;
    animation: breathe 5s infinite;
    background: var(--bg-card);
    padding: 2px 20px 6px 20px;
    border-radius: 8px;
  }
  @keyframes breathe {
    0% {
      background-color: #FFFFFF00;
      color: var(--text-color);
    }
    50% {
      background-color: var(--bg-card);
      color: var(--text-card-color);
    }
    100% {
      background-color: #FFFFFF00;
      color: var(--text-color);
    }
  }
</style>
