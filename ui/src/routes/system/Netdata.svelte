<script>
  import { structure } from '$lib/stores/websocket'
  import { page } from '$app/stores'
  import Fa from 'svelte-fa'
  import { faArrowUpRightFromSquare } from '@fortawesome/free-solid-svg-icons';
  export let wide
</script>

<div class="container {wide ? "wide" : "slim"}">
    <div class="title">Netdata</div>
    <a href="http://{$page.url.hostname}:19999" target="_blank" class="netdata">
      <Fa icon={faArrowUpRightFromSquare} size="2x" />
    </a>
</div>

<style>
  .wide {
    width: 992px;
    max-width: 98vw;
    padding: 56px;
  }
  .slim {
    width: 100vw;
  }
  .title {
    flex: 1;
    margin: 0;
  }
  .container {
    display: flex;
    gap: 20px;
  }
  a {
    color: var(--text-color);
  }
</style>
