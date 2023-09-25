<script>
  import { onMount, afterUpdate } from 'svelte'
  import { wide } from '$lib/stores/display';
  import { goto } from '$app/navigation';
  import { structure } from '$lib/stores/websocket';

  import Dropzone from './Dropzone.svelte';
  import NotFree from './NotFree.svelte';

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

  // debug
  $: upload = ($structure?.upload) || {}
  $: status = (upload?.status) || ""
  $: patp = (upload?.patp) || ""
  $: total = (upload?.total) || 0
  $: done = (upload?.done) || 0
  $: error = (upload?.error) || ""
  let uploaded = 0
</script>

<div id="card-wrapper" class="card-wrapper {wide ? "wide" : "slim"}">
  <div class="title-wrapper">
    <div class="title">IMPORT PIER</div>
  </div>
  {#if status.length < 1}
    <div class="warning">
      <div class="text">Warning</div>
      <div class="text">Make sure your pier is not running anywhere else or your <strong>pier will be corrupted</strong></div>
    </div>
    <Dropzone on:progress={e=>uploaded=e.detail} />
  {:else}
    <NotFree {status} name={patp} {total} {done} {error} {uploaded} />
  {/if}
</div>
<style>
  .wide {
    width: 1104px;
    max-width: 100vw;
  }
  .slim {
    width: calc(100vw - 40px);
  }
  .card-wrapper {
    background: var(--bg-base);
    margin: auto;
    width: calc(1173px - 160px);
    border-radius: 16px;
    flex-shrink: 0;
    padding: 80px;
  }
  .title-wrapper {
    overflow: hidden;
    height: 30px;
    margin-bottom: 56px;
  }
  .title {
    position: relative;
    top: -19px;
    color: #000;
    text-align: center;
    leading-trim: both;
    text-edge: cap;
    font-family: BPdotsUnicase;
    font-size: 48px;
    line-height: 47px;
    font-style: normal;
    font-weight: 700;
    letter-spacing: -2.88px;
    text-transform: uppercase;
  }
  .warning {
    display: flex;
    width: calc(621px - 64px);
    padding: 32px;
    flex-direction: column;
    align-items: flex-start;
    gap: 32px;
    border-radius: 8px;
    background: var(--NP_Yellow, #EDF02C);
    margin: auto;
    margin-bottom: 56px;
  }
  .warning > .text {
    color: #000;
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    letter-spacing: -1.44px;
  }
  strong {
    font-weight: 500;
  }
</style>
