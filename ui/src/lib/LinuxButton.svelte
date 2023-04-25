<script>
  import { afterUpdate, onMount } from 'svelte'
  import { noconn, api, linuxUpdate, updateState } from '$lib/api.js'
  import { page } from '$app/stores'

  let hide = false

  afterUpdate(()=> {
    hide = ($page.route.id == '/login')
  })

	// updateState loop
  const update = () => {
    if (!$noconn && ($page.route.id != '/device-update')) {
      fetch($api + '/linux/updates', {credentials: "include"})
      .then(raw => raw.json())
        .then(res => {
          console.log(res)
          updateState(res)
        })
      .catch(err => {
        if ((typeof err) == 'object') {
          updateState({status:'noconn'})
        }
      })
    }
    setTimeout(update, 10000)
	}

	// Start the update loop
	onMount(()=> {
    api.set("http://" + $page.url.hostname + ":27016")
		update()
	})

</script>

{#if $linuxUpdate}
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
