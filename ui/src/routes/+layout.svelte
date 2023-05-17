<script>
  import { connect, socketInfo } from "$lib/stores/websocket" 
  import { onMount } from 'svelte'
  import { page } from '$app/stores'

  onMount(()=> {
    const hostname = $page.url.hostname
    connect("ws://" + hostname + ":8000", $socketInfo)
  })
</script>

<div>
  <pre>
    {JSON.stringify($socketInfo)}
  </pre>
  <slot/>
</div>
