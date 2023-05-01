<script>
	import { onMount, onDestroy } from 'svelte'
  import { scale } from 'svelte/transition'
  import { page } from '$app/stores'

  import Fa from 'svelte-fa'
  import { faCheck } from '@fortawesome/free-solid-svg-icons'

  import { socketInfo } from '$lib/stores/websocket.js'

	//import { updateState, api, system, noconn, startram } from '$lib/api'
  import Logo from '$lib/Logo.svelte'
	import Card from '$lib/Card.svelte'

  import AnchorHeader from '$lib/AnchorHeader.svelte'
  import AnchorInformation from '$lib/AnchorInformation.svelte'
  import AnchorRegisterKey from '$lib/AnchorRegisterKey.svelte'
  import AnchorAdvanced from '$lib/AnchorAdvanced.svelte'

	let inView = false

  $: startram = ($socketInfo.system?.startram) || {}
  $: register = (startram?.register) || "no"
  $: container = (startram?.container) || "stopped"
  $: region = (startram?.region) || "us-east"
  $: regions = (startram?.regions) || ["us-east"]
  $: autorenew = (startram?.autorenew) || false
  $: expiry = (startram?.expiry) || 0
  $: endpoint = (startram?.endpoint) || "api.startram.io"
  $: restart = (startram?.restart) || "hide"
  $: cancel = (startram?.cancel) || "hide"
  $: advanced = (startram?.advanced) || false

	onMount(()=> inView = true)
  onDestroy(()=> inView = false)
	
</script>

{#if inView}
  <Card width="460px">

    <!-- Header -->
    <AnchorHeader wgReg={register == "yes"} wgRunning={container == "running"}>
      <Logo t='StarTram'/>
    </AnchorHeader>

    {#if register == "yes"}
      <AnchorInformation
        region={region}
        regions={regions}
        ongoing={autorenew}
        lease={expiry}
      />
    {/if}

    <!-- Register Key --
    <AnchorRegisterKey
      wgReg={$startram.wgReg}
      region={$startram.region}
      regions={$startram.regions}
    />

    <div class="sign-up">
      <a href="https://www.nativeplanet.io/startram" target="_blank">
        Need a startram registration key? Get one here!
      </a>
    </div>

    <!-- Advanced Options --
    <AnchorAdvanced wgReg={$startram.wgReg} wgRunning={$startram.wgRunning} />
    -->
  </Card>
{/if}

<style>
  .lease {
    padding-top: 20px;
    font-size: 12px;
  }
  .sign-up {
    margin-top: 12px;
    margin-left: 2px;
  }
  a {
    color: inherit;
    font-size: 12px;
    text-decoration: underline;
    cursor: pointer;
  }
</style>
