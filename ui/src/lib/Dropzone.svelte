<script>
  import { onMount } from 'svelte'
  import { api, isPatp } from '$lib/api'
  import Dropzone from "dropzone"
  import { sigil, stringRenderer } from '@tlon/sigil-js'
  import LinkButton from '$lib/LinkButton.svelte'

  let dzStatus = "free",
    curProgress = 0,
    totalSize = 0,
    uploadedAmount = 0,
    fileName = '',
    failed = false,
    failedText = "File is invalid",
    allowCancel = true

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
      dzStatus = 'uploading'
    }
    if (prog == 100) {
      dzStatus = 'processing'
      allowCancel = false
    }
    curProgress = prog
    totalSize = file.size
    fileName = file.name 
    uploadedAmount = sent
  }

  const onSuccess = (file,res) => {
    console.log("success:" + res)
    if (res == 200) {
      let name = file.name.split(".")[0]
      handleSuccess(name)
    } else if (res == 404) {
      window.location.href = "/login"
    } else {
      dzStatus = 'free'
      failed = true
      failedText = res
      setTimeout(()=>{
        failed = false
        failedText = "File is invalid"
        allowCancel = true 
      }, 2400)
    }
  }

  const handleSuccess = n => {
    fetch($api + '/urbit?urbit_id=' + n, {credentials:'include'})
			.then(raw => raw.json())
      .then(res => {
        if (res.name == n) {
          window.location.href = '/' + n
        } else {
          setTimeout(()=> handleSuccess(n), 1000)
        }
      })
			.catch(err => console.log(err))
  }

  const onError = (e) => {
    console.log("error:" + e)
    dzStatus = 'free'
    failed = true
    failedText = e
    setTimeout(()=>{
      failed = false
      failedText = "File is invalid"
      allowCancel = true 
    }, 2400)
  }

  // Dropzone params
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

</script>
<div id="dropper" class={dzStatus == "free" ? "drop" : "disabled"}>

  {#if dzStatus == 'free'}
    {#if failed}
      <span style="color: red;">{failedText}</span>
    {:else}
      Drop pier here to upload
    {/if}
  {/if}

  {#if dzStatus == 'uploading'}

    <div class="content">
      <div class="filename">Uploading {fileName}</div>
      <div class="bar-wrapper">
        <div class="bar" style="width:{curProgress}%"></div>

        <div class="uploaded">
          {#if totalSize > (1000 * 1000 * 1000)}
            {parseFloat((uploadedAmount / (1000 * 1000 * 1000)).toFixed(2))} GB / {(totalSize / (1000 * 1000 * 1000)).toFixed(2)} GB
          {:else}
            {parseFloat((uploadedAmount / (1000 * 1000)).toFixed(2))} MB / {parseFloat((totalSize / (1000 * 1000)).toFixed(2))} MB
          {/if}
        </div>

        <div class="percent">{curProgress.toFixed(0)}%</div>
      </div>
    </div>

  {/if}

  {#if dzStatus == 'processing'}

    <div class="content">
      <div class="filename processing">Processing {fileName}</div>
      <div class="bar-wrapper">
        <div class="bar" style="width:{curProgress}%"></div>

        <div class="uploaded processing">
          {#if totalSize > (1000 * 1000 * 1000)}
            {(totalSize / (1000 * 1000 * 1000)).toFixed(2)} GB
          {:else}
            {(totalSize / (1000 * 1000)).toFixed(2)} MB
          {/if}
        </div>

        <div class="percent processing">{curProgress.toFixed(0)}%</div>
      </div>
    </div>

  {/if}

</div>

{#if allowCancel}
  <div style="text-align:center;margin: 24px 0 12px 0">
    <LinkButton
      left={false}
      text="Cancel"
      src="/"
      disabled={false}
    />
  </div>
{/if}

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
