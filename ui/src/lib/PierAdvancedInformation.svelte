<script>
  import { socketInfo } from '$lib/stores/websocket.js'
  export let name
  export let running
  $: vere = ($socketInfo.urbits[name]?.vere?.version) || ""
</script>

{#if running && (vere.length > 1)}
  <div class="bg">
    <div class="option-title">Additional Information</div>
    <div class="text">vere version: {vere}</div>
	</div>
{/if}

<style>
  .bg {
    background: #0000001d;
    padding: 20px 0 20px 0;
    border-radius: 12px;
  }
  .text {
    font-size: 12px;
  }
</style>
