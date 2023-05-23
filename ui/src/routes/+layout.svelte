<script>
  import { login, connect, structure } from "$lib/stores/websocket" 
  import { onMount } from 'svelte'
  import { page } from '$app/stores'

  let password;

  onMount(()=> {
    const hostname = $page.url.hostname
    connect("ws://" + hostname + ":8000")
  })
</script>

{JSON.stringify($structure)}

<input bind:value={password}>
<button on:click={()=>login(password)}>Login</button>
<div>
  <slot/>
</div>
