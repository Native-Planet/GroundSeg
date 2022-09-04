<script>
  import { onMount } from 'svelte'
  import { url } from '/src/Scripts/server'
  import Dropzone from "dropzone"

  let isUploading = false, curProgress = 0, totalSize = 0, uploadedAmount = 0, fileName = 'dister-dister-dister-dister.zip'

  onMount(()=> {
    let myDropzone = new Dropzone("#dropper", {
      paramName: "file", // The name that will be used to transfer the file
      acceptedFiles: '.zip, .tar, .tgz, .gz',
      chunking: true,
      forceChunking: true,
      url: url + '/upload/pier',
      disablePreviews: true,
      uploadprogress: checkUpdate,
      success: onSuccess,
      maxFilesize: 11000000, // megabytes
      chunkSize: 50000000 // bytes
    })
  })

  const checkUpdate = (file,prog,sent) => {
    if (file.status === 'uploading') {
      isUploading = true
    }
    curProgress = prog
    totalSize = file.size
    fileName = file.name 
    uploadedAmount = sent
  }

  const onSuccess = (file,res) => {
    if (res == 200) {
      let name = file.name.split(".")[0]
      window.location.href="/" + name
    }
  }

  const stall = c => {
    if (c < 100) {
      return true
    }
    if (c = 100) {
      setTimeout(()=>{return false}, 1000)
    }
  }

</script>

<div id="dropper" class={isUploading ? "disabled" : "drop"}>
  {#if isUploading}
    <div class="content">
      <div class="filename">{curProgress < 100 ? "Uploading" : "Processing"} {fileName}</div>
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
  {:else}
    Drop pier here to upload
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

</style>
