<script>
  import Dropzone from "dropzone"
  import { onMount } from 'svelte'
  import { checkPatp } from '$lib/stores/patp';
  import { structure, uploadMetadata, freeUpload } from '$lib/stores/websocket'
  import Sigil from './Sigil.svelte';
  import { page } from '$app/stores'
  import { createEventDispatcher } from 'svelte';

  export let confirmed = false
  export let status
  export let size
  export let patp

  let percent = 0

  const dispatch = createEventDispatcher()
  const secret = "aaaaaaaa"

  /**********************
  |   DEFAULT VALUES    |
  **********************/

  $: patp = ($structure?.upload?.patp) || "Not chosen"
  /* Options for additional functionality during post upload */
  let remoteCheck = true
  let fixCheck = true
  /*  Uploader API address */
  $: addr = "http://" + $page.url.hostname + ":27016/upload"

  /*  Now, we intialize the dropzone during mount */
  onMount(()=> {
    let myDropzone = new Dropzone("#dropper", {
      /* Display */
      disablePreviews: true,
      /* HTTP */
      withCredentials: true,
      url: addr,
      /* Chunking */
      chunkSize: 50000000, // bytes
      chunking: true,
      forceChunking: true,
      retryChunks: true,
      retryChunksLimit: 5,
      /* File Settings */
      acceptedFiles: '.zip, .tar, .tgz, .gz',
      maxFilesize: 11000000, // megabytes
      paramName: setName, // The name that will be used to transfer the file
      /* Events */
      /* Accept: returns a done() if accepted */
      /* Success: Completed all chunks */
      /* Error: Something went wrong */
      accept: verifyFile,
      uploadprogress: handleProgress,
      success: onSuccess,
      error: onError,
  })})

  $: registered = ($structure?.profile?.startram?.info?.registered) || false
  $: running = ($structure?.profile?.startram?.info?.running) || false

  const setName = () => {
    let remote = remoteCheck ? "remote" :"local"
    let fixer = fixCheck ? "yes" : "no"
    if (registered && running) {
      return "pier-" + remote + "-" + fixer + "-" + secret
    } else {
    return "pier-local-" + fixer + "-" + secret
    }
  }

  const verifyFile = (f,done) => {
    const patp = f.name.split('.')[0]
    const size = f.size
    if (!checkPatp(patp)) { return }
    dispatch('drop',{"size":f.size,"patp":patp,"secret":secret})
    return loop(done).then(r=>{return r})
  } 

  const loop = async done => {
    while (!confirmed) {
      await new Promise(resolve => setTimeout(resolve, 250))
    }
    return done()
  }

  const handleProgress = (file,prog,sent) => {
    percent = prog
  }

  const onSuccess = (file,res) => {
  }

  const onError = (e) => {
    console.log("error", e)
  }
  // This is to give the button access to the dropper element
  const selectDropper = () => {
    freeUpload()
    document.getElementById('dropper').click();
  }


</script>

<div id="dropper"></div>
<div class="checkboxes">
  {#if registered && running}
    <div class="check-wrapper" on:click={()=>remoteCheck = !remoteCheck}>
      <div class="checkbox" class:highlight={remoteCheck}></div>
      <div class="check-label">Set to remote</div>
    </div>
  {/if}
  <div class="check-wrapper" on:click={()=>fixCheck = !fixCheck}>
    <div class="checkbox" class:highlight={fixCheck}></div>
    <div class="check-label">Fix common issues</div>
  </div>
</div>
<div class="upload">
  <div on:click={selectDropper} class="select">{patp}</div>
  <button on:click={selectDropper} class="choose-btn">Choose</button>
</div>
{#if (checkPatp(patp) && ((status != "free") || (status != "failed")))}
  <div class="ship">
    <Sigil
      {patp}
      padding="20px"
      {percent}
      size="120px"
      rad="16px"
      />
    <div class="info">
      <div class="item">Uploaded: {percent}%</div>
      <div class="item">File Size: {size}</div>
    </div>
  </div>
{/if}

<style>
  #dropper {
    display:none;
  }
  .upload {
    display: flex;
    gap: 24px;
    height: 48px;
    margin-bottom: 24px;
    width: 681px;
  }
  .select {
    flex: 1;
    border-radius: 16px;
    border: solid 2px var(--btn-secondary);
    color: var(--text-color);
    line-height: 48px;
    background: none;
    padding-left: 20px;
    font-size: 12px;
    padding-left: 24px;
  }
  .choose {
    display: none;
  }
  .choose-btn {
    border-radius: 16px;
    background-color: var(--btn-secondary);
    color: var(--text-card-color);
    padding: 0 48px;
    cursor: pointer;
  }
  .ship {
    display: flex;
    width: 60%;
  }
  .info {
    flex: 1;
    margin: 40px;
    display: flex;
    flex-direction: column;
  }
  .item {
    font-size: 24px;
  }
  .checkboxes {
    display: flex;
    gap: 48px;
  }
  .check-wrapper {
    cursor: pointer;
    user-select: none;
    display: flex;
    gap: 12px;
    align-items: start;
  }
  .checkbox {
    width: 20px;
    height: 20px;
    border: solid 1px var(--btn-secondary);
    border-radius: 6px;
  }
  .highlight {
    background-color: var(--btn-secondary);
  }
  .check-label {
    line-height: 20px;
    font-size: 12px;
    margin-bottom: 24px;
  }
</style>
