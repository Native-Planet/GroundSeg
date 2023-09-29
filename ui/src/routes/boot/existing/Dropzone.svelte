<script>
  import Dropzone from "dropzone"
  import { onMount, createEventDispatcher } from 'svelte'
  import { checkPatp } from '$lib/stores/patp'
  import { generateRandom } from '$lib/stores/gs-crypto'
  import { warningDone } from './store'
  import { wsPort, structure, modifyUploadEndpoint, openUploadEndpoint } from '$lib/stores/websocket'
  import Sigil from './Sigil.svelte'
  import { page } from '$app/stores'
  import { goto } from '$app/navigation'

  import { openModal } from 'svelte-modals'
  import WarningPrompt from './WarningPrompt.svelte'

  const endpoint = generateRandom(32)
  const dispatch = createEventDispatcher()

  /**********************
  |   DEFAULT VALUES    |
  **********************/

  let patp = ""
  let filename = ""
  let remote = true;
  let fix = true;
  let percent = 0

  /*  Uploader API address */
  $: addr = "http://" + $page.url.hostname + ":" + $wsPort + "/import/" + endpoint
  $: registered = ($structure?.profile?.startram?.info?.registered) || false
  $: running = ($structure?.profile?.startram?.info?.running) || false


  /*  Now, we intialize the dropzone during mount */
  onMount(()=> {
    let myDropzone = new Dropzone("#dropper", {
      /* Display */
      disablePreviews: true,
      /* HTTP */
      /*withCredentials: true,*/
      url: handleAddr,
      /* Chunking */
      chunkSize: 50000000, // bytes
      chunking: true,
      forceChunking: true,
      retryChunks: true,
      retryChunksLimit: 5,
      /* File Settings */
      acceptedFiles: '.zip, .tar, .tgz, .gz',
      maxFilesize: 11000000, // megabytes
      paramName: "file", // The name that will be used to transfer the file
      /* Events */
      /* Accept: call done() in accepted */
      /* Success: Completed all chunks */
      /* Error: Something went wrong */
      accept: handleUpload,
      uploadprogress: handleProgress,
      success: onSuccess,
      error: onError,
  })})

  const handleAddr = () => {
    const fullAddr = addr + "/" + patp
    return fullAddr
  }

  const handleUpload = (file, done) => {
    filename = file.name
    let p = filename.split(".")[0]
    let valid = checkPatp(p)
    if (valid) {
      patp = p 
    }
    openUploadEndpoint(endpoint,remote,fix)
    waitForWarning(done)
  }

  const waitForWarning = (done) => {
    if (!$warningDone) {
      setTimeout(()=>waitForWarning(done),200) 
    } else {
      warningDone.set(false)
      done()
    }
  }

  const handleProgress = (file,prog,sent) => {
    dispatch("progress",prog)
    percent = prog
  }

  const onSuccess = (file,res) => {
  }

  const onError = (e) => {
    console.log("error", e)
  }
  // This is to give the button access to the dropper element
  const selectDropper = () => {
    document.getElementById('dropper').click();
  }

  const handleRemote = () => {
    remote = !remote
    if (patp.length > 0) {
      openUploadEndpoint(endpoint,remote,fix)
    }
  }

  const handleFix = () => {
    fix = !fix
    if (patp.length > 0) {
      openUploadEndpoint(endpoint,remote,fix)
    }
  }


</script>

<div class="input-wrapper">
  <div class="label">Pier</div>
  <div class="upload">
    <div id="dropper"></div>
    <div on:click={selectDropper} class="select">{patp.length < 1 ? "Not chosen" : filename}</div>
    <button on:click={selectDropper} class="btn action-btn">Choose</button>
  </div>
  {#if registered && running}
  <div class="check-wrapper" on:click={handleRemote}>
    <div class="checkbox">
      {#if remote}
        <img class="checkmark" src="/checkmark.svg" alt="checkmark"/>
      {/if}
    </div>
    <div class="check-label">Set to remote</div>
  </div>
  {/if}
  <div class="check-wrapper" on:click={handleFix}>
    <div class="checkbox">
      {#if fix}
        <img class="checkmark" src="/checkmark.svg" alt="checkmark"/>
      {/if}
    </div>
    <div class="check-label">Update configuration if needed </div>
  </div>
  <div class="buttons">
    <button class="btn back" on:click={()=>goto('/boot')}>Back</button>
    <button class="btn action-btn" disabled={patp.length < 1} on:click={()=>openModal(WarningPrompt)}>Import</button>
  </div>
</div>

<style>
  #dropper {
    display:none;
  }
  /*
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
  */
  .input-wrapper {
    margin: auto;
    display: flex;
    width: 621px;
    padding-bottom: 0px;
    flex-direction: column;
    align-items: flex-start;
    gap: 16px;
    margin-bottom: 16px;
  }
  .label {
    color: var(--Gray-400, #5C7060);
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    letter-spacing: -1.44px;
  }
  .upload {
    display: flex;
    gap: 16px;
    width: 621px;
    margin: auto;
  }
  .select {
    flex: 1;
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    letter-spacing: -1.44px;
    border-radius: 16px;
    padding: 15px 22px 12px 22px;
    width: calc(100% - 48px);
    border: 2px solid var(--Gray-400, #5C7060);
    background: var(--bg-base);
    color: var(--text-color);
  }
  .check-wrapper {
    display: flex;
    justify-content: center;
    align-items: center;
    gap: 16px;
    cursor: pointer;
    user-select: none; /* Standard syntax */
    -webkit-user-select: none; /* Safari */
    -moz-user-select: none; /* Firefox */
    -ms-user-select: none; /* IE/Edge */
  }
  .checkbox {
    width: 28px;
    height: 28px;
    border-radius: 4px;
    border: 2px solid var(--Gray-200, #ABBAAE);
  }
  .checkmark {
    margin: 4px;
  }
  .check-label {
    color: #000;
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    letter-spacing: -1.44px;
  }
  .buttons {
    display: flex;
    gap: 16px
  }
  .btn {
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    line-height: 32px; /* 133.333% */
    letter-spacing: -1.44px;
    color: #FFF;
    height: 65px;
  }
  .back {
    font-family: var(--regular-font);
    color: var(--text-card-color);
    cursor: pointer;
    background-color: var(--btn-secondary);
    border-radius: 16px;
    padding: 0 48px;
  }
  .action-btn {
    font-family: var(--regular-font);
    color: var(--text-card-color);
    cursor: pointer;
    border-radius: 16px;
    background-color: var(--btn-primary);
    height: 65px;
    padding: 0 48px;
  }
  .action-btn:hover {
    background-color: var(--bg-card);
  }
  .action-btn:hover {
    background-color: var(--bg-card);
  }
  .action-btn:disabled {
    opacity: .6;
    pointer-events: none;
  }
</style>
