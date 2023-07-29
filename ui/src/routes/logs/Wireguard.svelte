<script>
  import { onMount, onDestroy } from 'svelte'
  import { connected, structure, openWireguardLog } from '$lib/stores/websocket'
  export let wide

  $: wgLogs =($structure?.logs?.containers?.wireguard) || {}
  $: stream = (wgLogs?.stream) || "open"
  $: logs = (wgLogs?.logs) || []

  const openLog = () => {
    if ($connected && (stream == "closed")) {
      openWireguardLog()
      }
    setTimeout(()=>openLog(),1000)
  }
  onMount(()=>openLog())

</script>

<div class="container {wide ? "wide" : "slim"}">
  {#if logs.length < 1}
    <pre>empty</pre>
  {/if}
  {#each logs as ln}
    <pre>{ln}</pre>
  {/each}
</div>

<style>
  .wide {
    width: 992px;
    max-width: 98vw;
    padding: 0 56px;
  }
  .slim {
    width: 100vw;
  }
  .container {
    max-height: 76vh;
    overflow-y: scroll;
  }
</style>
