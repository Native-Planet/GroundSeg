<script>
  import { onMount } from 'svelte'
  import Logo from '$lib/Logo.svelte'
	import Card from '$lib/Card.svelte'
  import PrimaryButton from '$lib/PrimaryButton.svelte'
  import LinkButton from '$lib/LinkButton.svelte'
	import { updateState } from '$lib/api'
  import UploadPierCheck from '$lib/UploadPierCheck.svelte'
  import Dropzone from '$lib/Dropzone.svelte'

	export let data
	updateState(data)

  let warningCheck = false
  let fileTypes = ['.zip','.tar','.tar.gz','.tgz']
  let allowCancel = true

  const hideCancel = () => allowCancel = false

  onMount(()=> {
    if (data['status'] == 404) {
      window.location.href = "/login"
    }
  })

</script>

<Card width="480px">
  <Logo t="Boot an existing Urbit ID"/>
  {#if warningCheck}

    <!-- If warning check has been completed -->
    <div class="subtitle">
      <div>Accepted Extensions:</div>
      {#each fileTypes as f}
        <div class="file-type">{f}</div>
      {/each}
    </div>

    <Dropzone on:full={hideCancel} />

    {#if allowCancel}
      <LinkButton
        top=24
        left={false}
        text="Cancel"
        src="/"
        disabled={false}
      />
    {/if}

  {:else}

    <!-- If warning check has not been completed -->
    <UploadPierCheck />

    <div class="buttons">
      <LinkButton text="Back" src="/" disabled={false} />
      <PrimaryButton
        on:click={()=>warningCheck = !warningCheck}
        standard="I understand"
        left={false} />
    </div>

  {/if}
</Card>

<style>
  .subtitle {
    font-weight: 700;
    font-size: .9em;
    padding-bottom: 24px;
    display: flex;
    margin-top: 12px;
    gap: 6px;
    align-items: center;
  }
  .file-type {
    background: #000;
    font-size: 10px;
    padding: 4px 8px 4px 8px;
    border-radius: 6px;
  }
  .buttons {
    display: flex;
    margin-top: 48px;
  }
</style>
