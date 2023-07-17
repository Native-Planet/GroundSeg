<script>
  import { api } from '$lib/api'
  import PrimaryButton from '$lib/PrimaryButton.svelte'

  import NetworkEth from '$lib/NetworkEth.svelte'
  import NetworkWifi from '$lib/NetworkWifi.svelte'

  export let ethOnly, connected

</script>

  <div class="network">
    <div class="network-title">Connectivity</div>
    <NetworkEth {ethOnly} />

    {#if !ethOnly}
      <NetworkWifi {connected} />
    {/if}

  </div>

<style>
  .network {
    background: #0000001d;
    padding: 20px 30px;
    border-radius: 8px;
    font-size: 18px;
    gap: 12px;
  }
  .network-title {
    font-size: 18px;
    padding-bottom: 8px;
  }
</style>
