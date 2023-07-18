<script>
  import { wide } from '$lib/stores/display';
  import { goto } from '$app/navigation';

  import Uploader from './Uploader.svelte';
  /*
  import { onMount } from 'svelte'
  import { page } from '$app/stores'
	import { api } from '$lib/api'

  import Logo from '$lib/Logo.svelte'
	import Card from '$lib/Card.svelte'
  import PrimaryButton from '$lib/PrimaryButton.svelte'
	import { updateState } from '$lib/api'
  import UploadPierCheck from '$lib/UploadPierCheck.svelte'
  import Dropzone from '$lib/Dropzone.svelte'
  import LinkButton from '$lib/LinkButton.svelte'

  let warningCheck = false
  let fileTypes = ['.zip','.tar','.tar.gz','.tgz']

  onMount(()=> api.set("http://" + $page.url.hostname + ":27016"))
  */

</script>

<div class="card-wrapper {wide ? "wide" : "slim"}">
  <div class="title">IMPORT PIER</div>
  <div class="warning">
    <div class="text">Warning</div>
    <div class="text">Make sure your pier is not running anywhere else or your <strong>pier will be corrupted</strong></div>
  </div>
  <Uploader />
</div>
<!--
{#if warningCheck}
  <Card width="480px">
    <Logo t="Boot an existing Urbit ID"/>
    <!-- If warning check has been completed --
    <div class="subtitle">
      <div>Accepted Extensions:</div>
      {#each fileTypes as f}
        <div class="file-type">{f}</div>
      {/each}
    </div>
    <Dropzone />
  </Card>
{:else}
  <Card width="480px">
    <Logo t="Boot an existing Urbit ID"/>
    <div style="display:{warningCheck ? "none" : "block"};">
      <!-- If warning check has not been completed --
      <UploadPierCheck /
      <div class="buttons">
        <LinkButton text="Back" src="/" disabled={false} />
        <PrimaryButton
          on:click={()=>warningCheck = true}
          standard="I understand"
          left={false} />
      </div>
    </div>
  </Card>
{/if}
      -->

<style>
  .wide {
    width: calc((288px * 3) + (80px * 2));
    max-width: 100vw;
  }
  .slim {
    width: calc(100vw - 40px);
  }
  .card-wrapper {
    background: var(--bg-base);
    border-radius: 16px;
    margin: auto;
    height: 70vh;
    padding: 40px;
    display:flex;
    gap: 40px;
    flex-direction: column;
    align-items: center;
  }
  .slim .card-wrapper {
    background: var(--bg-base);
    border-radius: 16px;
    margin: auto;
    height: 70vh;
    display:flex;
    gap: 40px;
    flex-direction: column;
    align-items: center;
  }
  .title {
    font-family: var(--title-font);
    font-size: 48px;
  }
  .warning {
    background: var(--bg-warning);
    padding: 32px;
    display: flex;
    flex-direction: column;
    gap: 32px;
    width: 621px;
    max-width: calc(100vw);
    border-radius: 8px;
  }
  .warning > .text {
    font-size: 24px;
  }
  /*
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
  */
</style>
