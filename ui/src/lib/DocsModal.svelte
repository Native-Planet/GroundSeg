<script>
  // Modal
  import Modal from '$lib/Modal.svelte'
  import { closeModal } from 'svelte-modals'

  export let isOpen
  export let info

  $: title = info?.title || "No Title Provided"
  $: description = info?.description || "No description provided."
  $: docName = info?.docName || null
  $: docURL = info?.docURL || null
</script>

<Modal>
  {#if isOpen}
  <div class="wrapper">
    <div class="header">{title}</div>
    <div class="name">{description}</div>
    <div class="documentation">
      Guide: {#if (docURL != null) && (docName != null)}<a class="docu-link" href={docURL} target="_blank">{docName}</a>{:else}No guide provided.{/if}
    </div>
  </div>
  {/if}
</Modal>

<style>
  .wrapper {
    padding: 32px;
  }
  .header {
    color: #000;
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    line-height: 48px; /* 200% */
    letter-spacing: -1.44px;
  }
  .name {
    color: var(--text-color, #313933);
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    line-height: 42px; /* 133.333% */
    letter-spacing: -1.44px;
    max-width: 460px;
    margin: 32px 0 16px 0;
  }
  .documentation {
    color: var(--text-color, #313933);
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    line-height: 42px; /* 133.333% */
    letter-spacing: -1.44px;
    max-width: 460px;
    margin: 32px 0 16px 0;
  }
  .documentation > a {
    color: var(--text-color, #313933);
    text-decoration: underline;
    font-weight: 600;
  }
</style>
