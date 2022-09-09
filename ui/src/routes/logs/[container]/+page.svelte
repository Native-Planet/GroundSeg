<script>
  import { page } from '$app/stores'
  import Logs from '$lib/Logs.svelte'
  let log = $page.params.container
</script>

<div class="wrapper">
  <Logs {log} />
</div>

<style>
  .wrapper {
    width: 800px;
    max-width: calc(100vw - 40px);
  }
</style>
