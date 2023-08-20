<script>
	import { afterUpdate, onMount, onDestroy } from 'svelte'
  import { scale } from 'svelte/transition'
  import { page } from '$app/stores'

  import Fa from 'svelte-fa'
  import { faCheck } from '@fortawesome/free-solid-svg-icons'

  import { send, socketInfo, socket } from '$lib/stores/websocket.js'
  import Logo from '$lib/Logo.svelte'
	import Card from '$lib/Card.svelte'
  import PrimaryButton from '$lib/PrimaryButton.svelte'

	let inView = false
  let buttonStatus = 'standard'

  $: update = ($socketInfo.updates?.linux?.update) || "updated"

	onMount(()=> inView = true)
  onDestroy(()=> inView = false)

  let ack = false
  afterUpdate(()=> {
    if (!(ack) && (update == "success")) {
      let payload = {
        "category": "updates",
        "payload": {"module": "linux", "action": "refresh"}
      }
      setTimeout(async ()=> {
        ack = await send($socket, $socketInfo, document.cookie, payload)
      }, 3000)
    }
  })

  const updateDevice = () => {
    let payload = {
      "category": "updates",
      "payload": {"module": "linux", "action": "update"}
    }
    send($socket, $socketInfo, document.cookie, payload)
  }
	
</script>

{#if inView}
  <Card width="460px">
    <Logo t='Device Update'/>
    <div class="linux-wrapper">
      <div class="text">
        Update the underlying operating system of your Native Planet Device.
      </div>
      {#if update == 'success'}
        <div class="success">Device update succeeded!</div>
      {:else if update == 'initializing'}
        <div class="loading">Requesting to update</div>
      {:else if update == 'command'}
        <div class="loading">Updating</div>
      {:else if update == 'restarting'}
        <div class="loading restarting">Device is restarting</div>
      {:else if update.includes('failure')}
        <div class="loading failure">Error: {update.split('\n')[1]}</div>
      {:else if (update == 'pending') || (update == 'updated')}
        <PrimaryButton 
          on:click={updateDevice}
          standard={update == "pending" ? "Update and Restart Device" : "No Updates, Run Anyways"}
        />
      {/if}
    </div>
  </Card>
{/if}

<style>
  .linux-wrapper {
    text-align: center;
  }
  .text {
    padding: 40px;
    font-size: 12px;
  }
  .loading {
    height: 30px;
    font-size: 12px;
    line-height: 30px;
    animation: breathe 2s infinite;
  }
  .restarting {
    color: orange;
  }
  .failure {
    color: red;
  }
  .success {
    height: 30px;
    font-size: 12px;
    line-height: 30px;
    color: lime;
  }
</style>
