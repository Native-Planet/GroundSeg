<script>
  import { scale } from 'svelte/transition'
  import { page } from '$app/stores'
  import { onMount } from 'svelte'

  import { api } from '$lib/api'
  import { structure } from '$lib/stores/websocket'

  import Fa from 'svelte-fa'
  import { faCheck } from '@fortawesome/free-solid-svg-icons'

  import Logo from '$lib/Logo.svelte'
	import NewPierButtons from '$lib/NewPierButtons.svelte'

	import Card from '$lib/Card.svelte'
  import KeyDropper from '$lib/KeyDropper.svelte'

  let remoteCheck = true
  let name = ''
  let key = ''
  let inView = false

  $: startram = ($structure?.system?.startram) || {}
  $: container = (startram?.container) || "stopped"
  $: register = (startram?.register) || "no"

  onMount(()=> {
    api.set("http://" + $page.url.hostname + ":27016")
    inView = !inView
  })

</script>

{#if inView}
<Card width="480px">
  <Logo t="Boot a new Urbit ID"/>
	<div class="key" in:scale={{duration:160, delay: 360}}>
  	<div class="info">
    	<div class="title">Urbit ID</div>
	    <input spellcheck="false" placeholder="sampel-palnet" bind:value={name}/>
  	</div>

	  <div class="info">
  	  <div class="title">Keyfile</div>
      <KeyDropper on:change={e=> key = e.detail} />
	  </div>

    <!-- Remote Autoset -->
    {#if register == "yes" && container == "running"}
      <div class="remote-check">
        <div class="box" class:highlight={remoteCheck} on:click={()=> remoteCheck = !remoteCheck}>
          {#if remoteCheck}
            <Fa icon={faCheck} size="1x"/>
          {/if}
        </div>
        <span on:click={()=> remoteCheck = !remoteCheck}>Automatically enable remote access</span>
      </div>
    {/if}

	</div>

  <NewPierButtons {name} {key} {remoteCheck} />

</Card>
{/if}

<style>
  input {
    flex: 1;
    padding: 8px;
    font-size: 12px;
    color: inherit;
    font-weight: 700;
    background: #ffffff4d;
    outline: none;
    border: none;
    border-radius: 6px;
  }
  ::-moz-placeholder {
    color: white;
  }
  ::-webkit-input-placeholder {
    color: white;
  }
  .key {
    display: flex;
    flex-direction: column;
    gap: 12px;
    color: inherit;
    padding: 20px;
    max-width: calc(100vw - 40px);
  }
  .info {
    display: flex;
    flex-direction: column;
  }
  .title {
    font-family: inherit;
    font-size: 13px;
    font-weight: 700;
    margin-bottom: 8px;
    text-align: left;
  }
  .remote-check {
    flex: 1;
    display: flex;
    gap: 6px;
    align-items: center;
    text-align: center;
    font-size: 11px;
    margin-top: 16px;
  }
  .box {
    width: 14px;
    height: 14px;
    background: #ffffff4d;
    border-radius: 4px;
    cursor: pointer;
    user-select: none;
  }
  span {
    cursor: pointer;
    user-select: none;
  }
  .highlight {
    background: #028AFB;
  }
</style>
