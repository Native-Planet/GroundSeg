<script>
  import { structure } from '$lib/stores/websocket'
  export let wide

  //$: logData = ($structure?.logs?.system) || ["empty"]
</script>

<div class="container {wide ? "wide" : "slim"}">
  <pre>yeetus</pre>
  <!--
  {#each logData as ln}
    <pre>{ln}</pre>
  {/each}
  -->
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
