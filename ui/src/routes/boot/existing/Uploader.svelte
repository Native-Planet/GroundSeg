<script>
  import { onMount } from 'svelte'
  import { Sha256 } from '@aws-crypto/sha256-browser';
  import { goto } from '$app/navigation';
  import { checkPatp } from '$lib/stores/patp';
  import { toBase64 } from '$lib/stores/gs-crypto'
  import { structure, freeUpload, uploadMetadata } from '$lib/stores/websocket'
  //import { processFile /*, manifest*/ } from '$lib/stores/uploader'

  import Dropzone from "dropzone"

  // Initialize the file object
  let file;
  let size;
  let patp;
  let currentChunk;
  let totalChunks;
  const chunkSize = 25 * 1024 * 1024; // 25MB

  $: patp = ($structure?.upload?.patp) || null
  $: status = ($structure?.upload?.status) || "free"
  $: svSize = ($structure?.upload?.size) || null
  $: svTotalChunks = ($structure?.upload?.totalChunks) || null
  //$: pending = ($structure?.upload?.pending) || {}

  // Read the size
  // save size in indexeddb
  // Send the size
  // Read the first 25MB
  // Hash it
  // Save hash in indexeddb
  // Send it
  // Send the hash
  // wait for hash response
  // delete hash

  /*
  const onFileChange = e => {
    file = e.target.files[0];
    size = file.size;
    patp = file.name.split('.')[0]
    currentChunk = 0
    totalChunks = Math.ceil(file.size / chunkSize);
    uploadMetadata(patp,size,totalChunks)
  }

  const clickInput = () => {
    freeUpload()
    document.getElementById('file-input').click();
  }

  const onSubmit = () => {
    if (file) {
      processChunk(file);
    } else {
      alert('Please select a file');
    }
  }

  const processChunk = f => {
    if (currentChunk >= totalChunks) {
      console.log("deal with completed stuff here")
      return
    }
    let blob = f.slice(chunkSize * currentChunk, chunkSize * (currentChunk + 1));
    const reader = new FileReader();
    reader.onload = async function(event) {
      let arrayBuffer = event.target.result
      const hash = await makeHash(arrayBuffer);
      const data = arrayBuffer
      saveAndSendChunk(hash,data,currentChunk,totalChunks)
      currentChunk++
      processChunk(f)
    }
    reader.readAsArrayBuffer(blob)
  }

  const makeHash = async arrayBuffer => {
    let sha = new Sha256()
    sha.update(arrayBuffer)
    const res = await sha.digest()
    return toBase64(res)
  }

  const saveAndSendChunk = (hash,data,currentChunk,totalChunks) => {
    sendChunk(hash,data,currentChunk,totalChunks)
  }
  */

</script>

<div class="wrapper">
  <div class="title">Pier</div>
  <Dropzone />
  <div class="action">
    <button on:click={()=>goto("/boot")} class="back">Back</button>
    <!--
    <button on:click={onSubmit} class="import">Import</button>
    -->
  </div>
</div>

<style>
  .wrapper {
    width: 681px;
    max-width: calc(100vw);
  }
  .title {
    margin-bottom: 10px;
  }
  .action {
    display: flex;
    height: 48px;
    gap: 24px;
  }
  .back {
    border-radius: 16px;
    background-color: var(--btn-secondary);
    color: var(--text-card-color);
    cursor: pointer;
    line-height: 48px;
    padding: 0 48px;
  }
  .import {
    border-radius: 16px;
    background-color: var(--btn-primary);
    color: var(--text-card-color);
    cursor: pointer;
    line-height: 48px;
    padding: 0 48px;
  }
</style>
