<script>
  import { afterUpdate } from 'svelte'
  import { page } from '$app/stores'
  import { socketInfo } from '$lib/stores/websocket.js'

  let hide = false
  afterUpdate(()=> {
    hide = ($page.route.id == '/login')
  })

  $: update = ($socketInfo.updates?.linux?.update) || "updated"

</script>

{#if update == "pending"}
  <a href='/device-update' class:hide={hide}>
    <div class="updates">Update Device</div>
  </a>
{/if}

<style>
  .hide {
    opacity: 0;
    pointer-events: none;
  }
	a {
		position:absolute;
		left: 150px;
    top: 3px;
	}
  .updates {
    display: flex;
    margin-top: 18px;
    font-size: 10px;
    line-height: 24px;
    padding: 0 8px;
    color: white;
    background: green;
    border-radius: 8px;
  }
</style>
