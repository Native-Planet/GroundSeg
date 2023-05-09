<script>
	import { onMount, onDestroy } from 'svelte'
  import { scale } from 'svelte/transition'
  import { page } from '$app/stores'

  import Fa from 'svelte-fa'
  import { faCheck } from '@fortawesome/free-solid-svg-icons'

  import { send, socket, socketInfo } from '$lib/stores/websocket.js'

  import Logo from '$lib/Logo.svelte'
	import Card from '$lib/Card.svelte'

  import AnchorHeader from '$lib/AnchorHeader.svelte'
  import AnchorInformation from '$lib/AnchorInformation.svelte'
  import AnchorRegisterKey from '$lib/AnchorRegisterKey.svelte'
  import AnchorAdvanced from '$lib/AnchorAdvanced.svelte'

	let inView = false

  $: startram = ($socketInfo.system?.startram) || null
  $: register = (startram?.register) || "no"
  $: container = (startram?.container) || "stopped"
  $: region = (startram?.region) || null
  $: regions = (startram?.regions) || []
  $: autorenew = (startram?.autorenew) || false
  $: expiry = (startram?.expiry) || null
  $: endpoint = (startram?.endpoint) || null
  $: restart = (startram?.restart) || "hide"
  $: cancel = (startram?.cancel) || "hide"
  $: advanced = (startram?.advanced) || false

  onMount(()=> inView = true)
  onDestroy(()=> inView = false)
	
</script>

{#if inView && (register != null)}
  <Card width="460px">
    <!-- Header -->
    <AnchorHeader wgReg={register == "yes"} wgRunning={container == "running"}>
      <Logo t='StarTram'/>
    </AnchorHeader>

    {#if register == "yes"}
      <AnchorInformation
        {region}
        {regions}
        ongoing={autorenew}
        lease={expiry}
      />
    {/if}

    <!-- Register Key -->
    <AnchorRegisterKey />

    <div class="sign-up">
      <a href="https://www.nativeplanet.io/startram" target="_blank">
        Need a startram registration key? Get one here!
      </a>
    </div>

    <!-- Advanced Options -->
    <AnchorAdvanced 
      {endpoint}
      wgReg={register == "yes"}
      wgRunning={container == "running"}
    />
  </Card>
{/if}

<style>
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
