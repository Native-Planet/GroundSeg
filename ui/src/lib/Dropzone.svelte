<script>
  import { onMount } from 'svelte'
  import { api, isPatp } from '$lib/api'
  import Dropzone from "dropzone"
  import { sigil, stringRenderer } from '@tlon/sigil-js'
  import { createEventDispatcher } from 'svelte';

  const dispatch = createEventDispatcher();

  let isUploading = false,
    curProgress = 0,
    totalSize = 0,
    uploadedAmount = 0,
    fileName = '',
    failed = false,
    failedText = "File is invalid"

  onMount(()=> {
    let myDropzone = new Dropzone("#dropper", {
      paramName: "file", // The name that will be used to transfer the file
      acceptedFiles: '.zip, .tar, .tgz, .gz',
      withCredentials: true,
      chunking: true,
      forceChunking: true,
      url: $api + '/upload',
      disablePreviews: true,
      uploadprogress: checkUpdate,
      success: onSuccess,
      error: onError,
      accept: checkPatp,
      maxFilesize: 11000000, // megabytes
      chunkSize: 50000000 // bytes
  })})

  const checkPatp = (f,done) => {
    let patp = f.name.split('.')[0]

    if (isPatp(patp)) {
      return done()
    } else { 
      failed = true
      setTimeout(()=>failed = false, 2400) 
  }}

  const checkUpdate = (file,prog,sent) => {
    if (file.status === 'uploading') {
      isUploading = true
    }
    if (prog == 100) {
      dispatch('full')
    }
    curProgress = prog
    totalSize = file.size
    fileName = file.name 
    uploadedAmount = sent
  }

  const onSuccess = (file,res) => {
    console.log(res)
    if (res == 200) {
      let name = file.name.split(".")[0]
      window.location.href = "/" + name
    } else if (res == 404) {
      window.location.href = "/login"
    } else {
      failed = true
      failedText = res
    }
  }

  const onError = (e) => {
    console.log(e)
    isUploading = false
    failed = true
    failedText = e
    setTimeout(()=>{failed = false}, 2400)
  }

</script>

<div id="dropper" class={isUploading ? "disabled" : "drop"}>
  {#if isUploading}
    <div class="content">
      <div class="filename" class:processing={curProgress == 100}>{curProgress < 100 ? "Uploading" : "Processing"} {fileName}</div>
      <div class="bar-wrapper">
        <div class="bar" style="width:{curProgress}%"></div>
        <div class="uploaded" class:processing={curProgress == 100}>
          {#if totalSize > (1000 * 1000 * 1000)}
            {#if curProgress == 100}
              {(totalSize / (1000 * 1000 * 1000)).toFixed(2)} GB
            {:else}
              {parseFloat((uploadedAmount / (1000 * 1000 * 1000)).toFixed(2))} GB / {(totalSize / (1000 * 1000 * 1000)).toFixed(2)} GB
            {/if}
          {:else}
            {#if curProgress == 100}
              {(totalSize / (1000 * 1000)).toFixed(2)} MB
            {:else}
              {parseFloat((uploadedAmount / (1000 * 1000)).toFixed(2))} MB / {parseFloat((totalSize / (1000 * 1000)).toFixed(2))} MB
            {/if}
          {/if}
        </div>
        <div class="percent" class:processing={curProgress == 100}>{curProgress.toFixed(0)}%</div>
      </div>
    </div>
  {:else}
    {#if failed}
      <span style="color: red;">{failedText}</span>
    {:else}
      Drop pier here to upload
    {/if}
  {/if}
</div>

<style>
  #dropper {
    height: 120px;
    width: 100%;
    text-align: center;
    margin: auto;
    cursor: pointer;
    position: relative;
  }
  .content {
    width: 100%;
    margin: 0;
    position: absolute;
    top: 50%;
    -ms-transform: translateY(-50%);
    transform: translateY(-50%);
    font-style: italic;
  }
  .filename {
    font-size: 14px;
  }
  .bar-wrapper {
    background: #ffffff4d;
    width: 80%;
    margin: auto;
    margin-top: 12px;
    height: 24px;
    position: relative;
    border-radius: 8px;
    overflow: hidden;
  }
  .bar {
    height: 100%;
    background: #040404;
    position: absolute;
    transition: width 400ms;
  }
  .uploaded {
    font-size: 12px;
    position: absolute;
    line-height: 24px;
    padding-left: 12px;
    left: 0;
  }
  .percent {
    font-size: 12px;
    position: absolute;
    right: 0;
    line-height: 24px;
    padding-right: 12px;
  }
  .drop {
    line-height: 120px;
    border: 1px solid #ffffff80;
  }
  .disabled {
    pointer-events: none;
  }
  .processing {
     animation: breathe 2s infinite;
  }

</style>
