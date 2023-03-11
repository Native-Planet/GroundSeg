<script>
	import { api } from '$lib/api'
  import Fa from 'svelte-fa'
  import { faRightLeft } from '@fortawesome/free-solid-svg-icons'

  import ExtUrl from '$lib/ExtUrl.svelte'
	import EyeButton from '$lib/EyeButton.svelte'
  import Clipboard from 'clipboard'

	export let name, urbitUrl, showUrbWeb, urbWebAlias

	let view = false, clicked = false
  let loading = false

	const toggleView = e => view = e.detail

  let copy = new Clipboard('#urbitUrl');
 	copy.on("success", ()=> {
  clicked = true; setTimeout(()=> clicked = false, 1000)})

  const swapUrl = () => {
		fetch($api + '/urbit?urbit_id=' + name, {
    method: 'POST',
    credentials: "include",
    headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({'app':'pier','data': 'swap-url'})
    })
      .then(d=>d.json())
      .then(r=> console.log(r))
      .catch(err => console.log(err))
      .then(()=> loading = false)
  }

</script>

<div class="pier-info">
  {#if urbWebAlias.length <= 0}
    <div class="pier-title">Ship Access URL</div>
  {:else}
    {#if showUrbWeb == 'alias'}
      <div class="pier-title">Custom Ship Access URL
        <button class="swap" class:loading={loading} on:click={swapUrl}>
          <Fa icon={faRightLeft} size="1.4x"/>
        </button>
      </div>
    {:else}
      <div class="pier-title">Default Ship Access URL
        <button class="swap" class:loading={loading} on:click={swapUrl}>
          <Fa icon={faRightLeft} size="1.4x"/>
        </button>
      </div>
    {/if}
  {/if}
  <div class="pier-cred-wrapper">
    <div on:click={copy} id="urbitUrl" data-clipboard-text={urbitUrl} class="pier-cred">
			{
				clicked ? "copied!" : view
				? urbitUrl : "click to copy"
      }
    </div>
		<ExtUrl link={urbitUrl} />
		<EyeButton on:click={toggleView} {view} />
  </div>
</div>

<style>
  .swap {
    padding: 0 8px 0 8px;
    color: inherit;
  }
  .swap:hover {
    cursor: pointer;
  }
  .loading {
    opacity: 0.6;
  }
</style>
