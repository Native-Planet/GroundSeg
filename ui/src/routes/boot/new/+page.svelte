<script>
  import { scale } from 'svelte/transition'
	import { onMount } from 'svelte'
	import { updateState } from '$lib/api'
  import Logo from '$lib/Logo.svelte'
	import Card from '$lib/Card.svelte'
	import NewPierButtons from '$lib/NewPierButtons.svelte'

	export let data
	updateState(data)

  let name = '', key = '', viewKey = false, inView = false

  const toggleViewKey = () => {
    viewKey = !viewKey
    document.querySelector('#key').type = viewKey ? 'text' : 'password'
  }

	onMount(()=> inView = !inView)

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
  	  <div class="title">Arvo Key</div>
    	<div class="pass-wrapper">
	      <input spellcheck="false" id="key" type="password" bind:value={key}/>
  	    <img on:click={toggleViewKey} src="/eye-{viewKey ? "closed" : "open"}.svg" alt="eye" />
    	</div>
	  </div>

	</div>

	<NewPierButtons {name} {key}/>

</Card>
{/if}

<style>
  .key {
    display: flex;
    flex-direction: column;
    gap: 24px;
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
  .pass-wrapper {
    display: flex;
  }
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
  .pass-wrapper > img {
    padding-left: 12px;
  }

</style>
