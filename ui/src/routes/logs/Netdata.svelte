<script>
  import { structure } from '$lib/stores/websocket'
  import { page } from '$app/stores'
  import Fa from 'svelte-fa'
  import { faArrowUpRightFromSquare } from '@fortawesome/free-solid-svg-icons';
</script>

<div class="container">
    <div class="title">NETDATA</div>
    <a href="http://{$page.url.hostname}:19999" target="_blank" class="netdata">
      <Fa icon={faArrowUpRightFromSquare} size="2x" />
    </a>
</div>

<style>
  .title {
    flex: 1;
    margin: 0;
  }
  .container {
    display: flex;
    margin: 0;
  }
  a {
    color: var(--text-color);
  }
</style>
