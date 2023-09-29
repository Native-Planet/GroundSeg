<script>
  import { connected, structure } from '$lib/stores/websocket'
  export let wide

  $: logs =($structure?.logs?.containers?.wireguard?.logs) || []

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
