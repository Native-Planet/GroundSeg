<script>
  import { url } from '/src/Scripts/server'
  import { page } from '$app/stores';
  import { onMount } from 'svelte'

  let log = $page.params.container
  let stream = []

  onMount(() => {
    const u = url + "/settings/logs"
    const f = new FormData()
    f.append('logs', log)
    fetch(u, {method: 'POST', body: f})
      .then(r => r.json()).then(d => {
        stream = d.split("\n")
      })
  })
</script>

{#each stream as s}
  <div>{s}</div>
{/each}

<style>
  div {
    margin-bottom: 4px;
    font-size: 12px;
    font-family:Consolas,Monaco,Lucida Console,Liberation Mono,DejaVu Sans Mono,Bitstream Vera Sans Mono,Courier New;
    width: 800px;
    padding: 0 20px 0 20px;
    max-width: calc(100vw - 40px);
  }
</style>
