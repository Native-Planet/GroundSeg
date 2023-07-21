<script>
  import { wide } from '$lib/stores/display';
  import { goto } from '$app/navigation';
  import { uploadMetadata, structure } from '$lib/stores/websocket';

  import Dropzone from './Dropzone.svelte';
  import WarningPrompt from './WarningPrompt.svelte';

  let showPrompt = false
  let confirmed = false

  const setPrompt = e => {
    showPrompt = true
    const patp = e.detail.patp
    const size = e.detail.size
    const secret = e.detail.secret
    uploadMetadata(patp,size,secret)
  }
  const confirm = () => {
    showPrompt = false
    confirmed = true
  }

  $: upload = ($structure?.upload) || {}
  $: patp = (upload?.patp) || null
  $: size = (upload?.size) || 0
  $: status = (upload?.status) || "free"
  $: uploaded = (upload?.uploaded) || 0

</script>

<div class="card-wrapper {wide ? "wide" : "slim"}">
  <div class="title">IMPORT PIER</div>
  <div class="warning">
    <div class="text">Warning</div>
    <div class="text">Make sure your pier is not running anywhere else or your <strong>pier will be corrupted</strong></div>
  </div>
  <Dropzone
    {size}
    {patp}
    {status}
    {confirmed}
    on:drop={setPrompt}
    />
</div>
{#if showPrompt}
  <WarningPrompt
    {status}
    on:confirm={confirm}
    />
{/if}

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
    display:flex;
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
    padding-top: 40px;
    padding-bottom: 40px;
  }
  .warning {
    background: var(--bg-warning);
    padding: 32px;
    margin-bottom: 40px;
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
</style>
