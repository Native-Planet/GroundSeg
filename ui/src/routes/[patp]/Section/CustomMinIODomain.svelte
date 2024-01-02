<script>
  // Style
  import "../theme.css"
  import { setMinIODomain, structure } from '$lib/stores/websocket'
  import { onMount, createEventDispatcher, afterUpdate } from 'svelte'
  import DocsModal from '$lib/DocsModal.svelte'
  import { openModal } from 'svelte-modals'
  export let patp
  export let minioAlias
  let domain = ""

  const dispatch = createEventDispatcher()

  $: tMinioDomain = ($structure?.urbits?.[patp]?.transition?.minioDomain) || ""
  $: t = tMinioDomain

  onMount(()=>domain = minioAlias)
  afterUpdate(()=> {
    if (t == "done") {
      dispatch("done")
    }
  })

  let docsInfo = {
    title: "Custom MinIO Domain",
    description: "Publish locally hosted media from custom domain.",
    docName: "Custom StarTram Domains",
    docURL: "https://manual.groundseg.app/guide/custom-domains.html"
  }

</script>

<div>
  <div class="section-title-wrapper">
    <div class="section-title">Custom MinIO Domain</div>
    <div class="what" on:click={()=>openModal(DocsModal, {info:docsInfo})}>?</div>
  </div>
  <div class="wrapper">
    <input type="text" placeholder="minio.example.com" bind:value={domain} disabled={t.length > 0} />
    <button disabled={(domain.length < 1) || (domain == minioAlias) || (t.length > 0)} class="save-button" on:click={()=>setMinIODomain(patp, domain)}>
      {#if t.length < 1}
        Save
      {:else if t == "loading"}
        Loading..
      {:else if t == "error"}
        Error
      {:else if t == "success"}
        Success!
      {/if}
    </button>
  </div>
</div>

<style>
  .section-title-wrapper {
    display: flex; 
    align-items: center;
    gap: 16px;
  }
  .wrapper {
    display: flex;
    gap: 16px;
    margin: 16px 0;
  }
  input {
    flex: 1;
    align-items: center;
    border-radius: 16px;
    color: var(--Gray-200, #ABBAAE);
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 32px;
    font-style: normal;
    font-weight: 300;
    line-height: normal;
    letter-spacing: -1.92px;
    padding: 14px;
    border: 2px solid var(--text-card-color);
    background: var(--bg-card);
  }
  .save-button {
    padding: 20px 48px;
    justify-content: center;
    align-items: center;
    gap: 8px;
    border-radius: 16px;
    background: #2C3A2E;
    color: var(--Gray-200, #ABBAAE);
    font-family: Inter;
    font-size: 32px;
    font-style: normal;
    font-weight: 300;
    line-height: 24px; /* 75% */
    letter-spacing: -1.92px;
    cursor: pointer;
  }
  .save-button:disabled {
    opacity: .6;
    pointer-events: none;
  }
  .what {
    width: 20px;
    height: 20px;
    text-align: center;
    border: 1px solid #FFF;
    border-radius: 50%;
    cursor: pointer;
    font-size: 16px;
  }
  .what:hover {
    opacity: .2;
  }
  input:disabled {
    opacity: .6;
    pointer-events: none;
  }
</style>
