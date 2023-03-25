<script>
  import { onMount } from 'svelte'
  import { api, isPatp } from '$lib/api'
  import { sigil, stringRenderer } from '@tlon/sigil-js'
  import Fa from 'svelte-fa'
  import { faCheck } from '@fortawesome/free-solid-svg-icons'

  import Dropzone from "dropzone"
  import LinkButton from '$lib/LinkButton.svelte'

  let dzStatus = "free"
  let curProgress = 0
  let totalSize = 0
  let uploadedAmount = 0
  let fileName = ''
  let failed = false
  let failedText = "File is invalid"
  let allowCancel = true
  let statuses = new Set(['uploaded'])
  let showStatuses = []
  let current = ''
  let extractProg = {}

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
      let patp = file.name.split('.')[0]
      getUploadStatus(patp,'status')
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
      console.log("handle success: " + name)
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

  const getUploadStatus = (n,act) => {
    fetch($api + '/upload/progress', {
			method: 'POST',
      credentials: "include",
			headers: {'Content-Type': 'application/json'},
			body: JSON.stringify({'patp': n,'action': act })
	  })
			.then(raw => raw.json())
      .then(res => {
        console.log(res)
        showStatuses = Array.from(statuses)
        if (res.status == 'removed') {
          current = ''
        } else if (res.status == 'done') {
          current = ''
        } else if (res.status == 'none') {
          if (!(showStatuses.includes('done'))) {
            current = ''
            dzStatus = 'free'
            failed = true
            failedText = "Unable to get progress"
            setTimeout(()=>{
              failed = false
              failedText = "File is invalid"
              allowCancel = true 
            }, 2400)
          }
        } else if (res.status == 'extracting') {
          statuses.add(res.status)
          current = res.status
          extractProg = res.progress
          setTimeout(()=>getUploadStatus(n,act), 1000)
        } else if (res.status == 'uploading') {
          setTimeout(()=>getUploadStatus(n,act), 1000)
        } else {
          statuses.add(res.status)
          current = res.status
          setTimeout(()=>getUploadStatus(n,act), 1000)
        }
      })
    .catch(err => console.log(err))
  }

  const handleSuccess = n => {
    getUploadStatus(n, 'remove')
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
      Drop Pier File here to Upload
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

    <div class="processing">
      <div class="processing-filename">Processing {fileName}</div>
      <div class="statuses">
        {#each showStatuses as s}
          <div class="status-wrapper">
            {#if s != 'done'}
              <div class="icon">
                {#if (s != current)}
                  <Fa icon={faCheck} size="1x" />
                {:else}
                  <div class="loader"></div>
                {/if}
              </div>
            {/if}
            {#if s == 'uploaded'}
              <div class="status-step">uploaded</div>
            {/if}
            {#if s == 'setup'}
              {#if s == current}
                <div class="status-step">getting environment ready</div>
              {:else}
                <div class="status-step">environment ready</div>
              {/if}
            {/if}
            {#if s == 'extracting'}
              {#if s == current}
                <div class="status-step">extracting ({(extractProg.current / extractProg.total * 100).toFixed(2)}%)</div>
              {:else}
                <div class="status-step">extracted</div>
              {/if}
            {/if}
            {#if s == 'cleaning'}
              {#if s == current}
                <div class="status-step">cleaning upload environment</div>
              {:else}
                <div class="status-step">upload environment cleaned</div>
              {/if}
            {/if}
            {#if s == 'booting'}
              {#if s == current}
                <div class="status-step">setting up your Urbit ship</div>
              {:else}
                <div class="status-step">Urbit ship is ready. Redirecting...</div>
              {/if}
            {/if}
          </div>
        {/each}
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
    min-height: 120px;
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
  .processing {
    width: 100%;
    margin: auto;
    height: 160px;
  }
  .processing-filename {
    font-size: 16px;
    margin-bottom: 8px;
  }
  .statuses {
    text-align: left;
    font-size: 13px;
    margin: auto;
    margin-top: 18px;
    width: 60%;
    display: flex;
    flex-direction: column;
    gap: 6px;
  }
  .status-wrapper {
    display: flex;
    gap: 4px;
  }
  .status-step {
    font-size: 12px;
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
  .icon {
    width: 16px;
    color: lime;
  }
  .loader {
    margin: 3px;
    border: 1px solid transparent;
    border-top: 1px solid white;
    border-bottom: 1px solid white;
    border-radius: 50%;
    width: 8px;
    height: 8px;
    animation: spin 1s linear infinite;
  }
  @keyframes spin {
    0% { transform: rotate(0deg); }
    100% { transform: rotate(360deg); }
  }
</style>
