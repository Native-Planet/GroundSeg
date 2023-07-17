<script>
  import { connect, structure, connected } from "$lib/stores/websocket" 
  import { onMount } from 'svelte'
  import { page } from '$app/stores'

  onMount(()=> {
    const hostname = $page.url.hostname
    connect("ws://" + hostname + ":8000")
    redirector()
  })

  $: access = ($structure?.system?.login?.access) || "unauthorized"

  let count = 0
  const redirector = () => {
    if ($connected) {
      const auth = (access === "authorized")
      if (auth) {
        if ($page.route.id === "/login") {
          window.location.href = "/"
        }
      } else {
        if (access === "unauthorized") {
          if ($page.route.id !== "/login") {
            if (count > 2) {
              count = 0
              window.location.href = "/login"
            } else {
              count += 1 
            }
          }
        }
      }
    }
    setTimeout(redirector,500)
  }

</script>
{#if !$connected}
  <div class="ws">connecting</div>
{/if}

<style>
  .ws {
    position: absolute;
    right: 0;
    bottom: 0;
    width: auto;
    height: auto;
    color: orange;
    font-size: 12px;
    margin: 12px;
  }
</style>
