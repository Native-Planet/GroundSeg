<script>
  import { connected } from '$lib/stores/websocket';
</script>
{#if !$connected}
  <div class="text">CONNECTING</div>
{/if}
<style>
  .text {
    position: absolute;
    bottom: 20px;
    right: 20px;
    font-family: var(--title-font);
    color: var(--text-color);
    font-size: 20px;
    animation: breathe 3s infinite;
  }
  @keyframes breathe {
    0% {opacity: .2}
    50% {opacity: 1}
    100% {opacity: .2}
  }
</style>
