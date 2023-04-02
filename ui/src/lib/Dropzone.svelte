<script>
  import { onMount } from 'svelte'
  import { api, isPatp, startram } from '$lib/api'
  import Fa from 'svelte-fa'
  import { faCheck, faRotate } from '@fortawesome/free-solid-svg-icons'

  import Dropzone from "dropzone"
  import LinkButton from '$lib/LinkButton.svelte'

  // Remote
  let remoteCheck = true

  // Failure text
  let failed = ""

  // Total progress information
  let working = false
  let statuses = new Set([])
  let showStatuses = []
  let current = ""
  let extractProg = {}

  // Uploading information
  let curProgress = 0
  let totalSize = 0
  let uploadedAmount = 0
  let fileName = ""

  // Verify filename
  const checkPatp = (f,done) => {
    let patp = f.name.split('.')[0]
    if (isPatp(patp)) {
      return done()
    } else { 
      failed = patp + " is not a valid name!"
    }
  }

  let notChecked = true
  const checkUpdate = (file,prog,sent) => {
    working = true
    if (file.status === "uploading") {
      statuses.add("uploading")
      current = "uploading"
      showStatuses = Array.from(statuses)
    }
    curProgress = prog
    totalSize = file.size
    fileName = file.name 
    uploadedAmount = sent

    if (prog.toFixed(0) == 100) {
      if (notChecked) {
        let patp = file.name.split('.')[0]
        getUploadStatus(patp, 'status')
        notChecked = false
      }
    }
  }

  const onSuccess = (file,res) => {
    if (res == 200) {
      let patp = file.name.split('.')[0]
      handleSuccess(patp)
    } else if (res == 404) {
      window.location.href = "/login"
    } else {
      working = false
      failed = res
      statuses = new Set([])
    }
  }

  const onError = (e) => {
    working = false
    failed = e
    statuses = new Set([])
  }

  const handleSuccess = n => {
    fetch($api + '/urbit?urbit_id=' + n, {credentials:'include'})
      .then(raw => raw.json())
      .then(res => {
        console.log(res)
        if (res.name == n) {
          getUploadStatus(n, 'remove')
          window.location.href = '/' + n
        } else {
          setTimeout(()=> handleSuccess(n), 1000)
        }
    })
    .catch(err => console.log(err))
  }

  let err_count = 0
  const getUploadStatus = (n,act) => {
    if (working) {
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
            err_count = 0
            current = ''

          } else if (res.status == 'done') {
            err_count = 0
            current = ''

          } else if (res.status == 'none') {
            if (!(showStatuses.includes('done'))) {
              if (err_count < 5) {
                ++err_count
                setTimeout(()=>getUploadStatus(n, act), 1000)
              } else {
                current = ''
                failed = "Unable to get progress"
              }
            } else {
              err_count = 0
              current = ''
            }

          } else if (res.status == 'extracting') {
            statuses.add(res.status)
            current = res.status
            extractProg = res.progress
            err_count = 0
            setTimeout(()=>getUploadStatus(n,act), 1000)

          } else if (res.status == 'uploading') {
            err_count = 0
            setTimeout(()=>getUploadStatus(n,act), 1000)

          } else {
            statuses.add(res.status)
            current = res.status
            err_count = 0
            setTimeout(()=>getUploadStatus(n,act), 1000)
          }
        })
      .catch(err => console.log(err))
    }
  }

  // Dropzone params
  onMount(()=> {
    let myDropzone = new Dropzone("#dropper", {
      paramName: nameFile,//"file", // The name that will be used to transfer the file
      acceptedFiles: '.zip, .tar, .tgz, .gz',
      withCredentials: true,
      chunking: true,
      forceChunking: true,
      retryChunks: true,
      retryChunksLimit: 5,
      url: $api + '/upload',
      disablePreviews: true,
      uploadprogress: checkUpdate,
      success: onSuccess,
      error: onError,
      accept: checkPatp,
      maxFilesize: 11000000, // megabytes
      chunkSize: 50000000 // bytes
  })})

  const nameFile = () => {
    if ($startram.wgReg && $startram.wgRunning) {
      return "file-" + remoteCheck
    } else {
      return "file-false"
    }
  }

</script>

<!-- Remote Autoset -->
{#if $startram.wgReg && $startram.wgRunning}
  <div class="remote-check" class:freeze={working}>
    <div class="box" class:highlight={remoteCheck} on:click={()=> remoteCheck = !remoteCheck}>
      {#if remoteCheck}
        <Fa icon={faCheck} size="1x"/>
      {/if}
    </div>
    <span on:click={()=> remoteCheck = !remoteCheck}>Automatically enable remote access</span>
  </div>
{/if}

<div class="wrapper">

  {#if failed.length > 0}
    <button class="okay" on:click={()=>failed = ""}><Fa icon={faRotate} size="1x" /></button>
  {/if}
  <!-- Main dropper area -->
  <div id="dropper" class:failure={failed.length > 0} class:dz-border={!working} class:dz-text={!working}>
    {#if !working}
      {failed.length > 0 ? failed : "drop pier file here to upload"}
    {:else}
      <div class="working-text">{fileName}</div>
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
            {#if s == 'uploading'}
              {#if s == current}
                <div class="status-step">uploading - {curProgress.toFixed(0)}%
                  (
                  {#if totalSize > Math.pow(1024, 3)}
                    {parseFloat((uploadedAmount / Math.pow(1024, 3)).toFixed(2))} / {(totalSize / Math.pow(1024, 3)).toFixed(2)} GiB
                  {:else}
                    {parseFloat((uploadedAmount / Math.pow(1024, 2)).toFixed(2))} / {parseFloat((totalSize / Math.pow(1024, 2)).toFixed(2))} MiB
                  {/if}
                  )
                </div>
              {:else}
                <div class="status-step">uploaded</div>
              {/if}
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
                <div class="status-step">extracting - {(extractProg.current / extractProg.total * 100).toFixed(2)}%</div>
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
    {/if}
  </div>

  <!-- Cancel Button -->
  {#if !working}
    <div class="cancel-wrapper">
      <LinkButton left={false} text="Cancel" src="/" disabled={false} />
    </div>
  {/if}

</div>

<style>
  .wrapper {
    height: 180px;
    width: 100%;
    margin: auto;
  }
  #dropper {
    box-sizing: border-box;
    width: 100%;
    margin-bottom: 20px;
    cursor: pointer;
    border: solid 1px none;
  }
  .okay {
    width: 100%;
    text-align: right;
    cursor: pointer;
    color: white;
    font-family: inherit;
    margin: 0 8px 8px 0;
  }
  .dz-text {
    line-height: 120px;
    text-align: center;
  }
  .dz-border {
    border: solid 1px white;
  }
  .failure {
    color: #BD4140;
    border: solid 1px #BD4140;
    pointer-events: none;
  }
  .cancel-wrapper {
    text-align:center;
    margin: 24px 0 12px 0;
  }
  .working-text {
    text-align: center;
    pointer-events: none;
  }
  .statuses {
    pointer-events: none;
    text-align: left;
    font-size: 13px;
    margin: auto;
    margin-top: 18px;
    width: 60%;
    display: flex;
    flex-direction: column;
    gap: 6px;
    height: 120px;
  }
  .status-wrapper {
    display: flex;
    gap: 4px;
  }
  .status-step {
    font-size: 12px;
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
  .remote-check {
    flex: 1;
    display: flex;
    gap: 6px;
    align-items: center;
    text-align: center;
    font-size: 11px;
    margin: 0 0 16px 4px;
  }
  .box {
    width: 14px;
    height: 14px;
    background: #ffffff4d;
    border-radius: 4px;
    cursor: pointer;
    user-select: none;
  }
  span {
    font-size: 12px;
    cursor: pointer;
    user-select: none;
  }
  .highlight {
    background: #028AFB;
  }
  .freeze {
    opacity: .6;
    pointer-events: none;
  }
</style>
