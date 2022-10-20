<script>
  import { api } from '$lib/api'
  import Clipboard from 'clipboard'

  import Fa from 'svelte-fa'
  import { faTriangleExclamation } from '@fortawesome/free-solid-svg-icons'
  import { faCircleQuestion } from '@fortawesome/free-regular-svg-icons'

  import PrimaryButton from '$lib/PrimaryButton.svelte'

	import PierCode from '$lib/PierCode.svelte'
	import PierUrl from '$lib/PierUrl.svelte'

	import PierNetwork from '$lib/PierNetwork.svelte'

  export let name, remote, wgReg, wgRunning, urbitUrl, code//, minIO, minIO_reg

					/*
  let viewLogin = false, clickedLogin = false,
    viewMinIO = false, clickedMinIO = false,
    minioPassword = '', viewKey = false,
    viewConfirm = false, confirmPassword = '' , showMinIOInfo = false,
    buttonStatus = 'failure', submitted = false, showButton = true, textToggle = 'none'

  // Toggle Password Visibility
  const toggleViewKey = () => {
    viewKey = !viewKey
    document.querySelector('#minio-password').type = viewKey ? 'text' : 'password'
  }

  const toggleViewConfirm = () => {
    viewConfirm = !viewConfirm
    document.querySelector('#minio-password-1').type = viewConfirm ? 'text' : 'password'
  }

  const handleTextToggle = s => {
    if (s == textToggle) {textToggle = 'none'}
    else {textToggle = s}
  }

  // Submit MinIO pasword
  const submitPassword = () => {
    let u = $api + "/urbit/minio/register"
    const f = new FormData()
    f.append('patp', name)
    f.append('password', confirmPassword)

    submitted = true
    buttonStatus = 'loading'

    fetch(u, {method: 'POST',body: f})
      .then(r => r.json())
      .then(d => { 
        if (d == 200) {buttonStatus = 'success'}
        else {buttonStatus = 'failure'}
        setTimeout(()=>showButton = false, 10000)
      })}


  // Copy String to Clipboard

  let copyLogin = new Clipboard('#login');
  let copyMinIO = new Clipboard('#minio');

  copyLogin.on("success", ()=> {
    clickedLogin = true; setTimeout(()=> clickedLogin = false, 1000)})

  copyMinIO.on("success", ()=> {
    clickedMinIO = true; setTimeout(()=> clickedMinIO = false, 1000)})

*/

</script>
	<!-- Landscape +code -->
	<PierCode {code} />

	<!-- Urbit Landscape URL -->
	<PierUrl {urbitUrl} />

		<!--
    {#if wg_reg && wg_running}

      <!-- Request for minIO password if not registered --
      {#if !minIO_reg}
        <div class="info">
          <div class="setup-title">
            <span>Setup MinIO Password</span>
            <button class="question-mark" on:click={()=>handleTextToggle('info')} >
              <Fa icon={faCircleQuestion} size="1.2x" />
            </button>
            <button class="alert-mark" on:click={()=>handleTextToggle('alert')} >
              <Fa icon={faTriangleExclamation} size="1.2x" />
            </button>
          </div>

          {#if textToggle == 'info' }
            <div class="minio-info">
              Store and share files on Urbit with MinIO. All data is stored locally on your device.
            </div>
          {/if}
          {#if textToggle == 'alert' }
            <div class="minio-info s3-alert">Warning: if you switch between anchors, it will break your previous S3 links.</div>
          {/if}
          
          {#if (minioPassword.length > 0) && (minioPassword.length < 8)}
            <div class="title-smaller">Password must have at least 8 characters</div>
          {/if}

          <div class="login-key-wrapper">
            <input
              id="minio-password"
              bind:value={minioPassword}
              class="minio-password"
              type="password"
              placeholder="Create a password to use MinIO" />
            <button on:click={toggleViewKey}>
              <img class="eye" src={viewKey ? "/eye-closed.svg" : "/eye-open.svg"} alt="eye" />
            </button>
          </div>
        </div>
      {/if}

  
        <!-- Confirm Password --
        {#if (minioPassword.length > 7) && !minIO_reg}
          <div class="info">
            <div class="title-smaller">Confirm Password</div>
  
            <div class="login-key-wrapper">
              <input
                id="minio-password-1"
                bind:value={confirmPassword}
                class="minio-password"
                type="password"
                placeholder="Enter the password again" />
              <button on:click={toggleViewConfirm}>
                <img class="eye" src={viewConfirm ? "/eye-closed.svg" : "/eye-open.svg"} alt="eye" />
              </button>
            </div>
          </div>
        {/if}

  
        <!-- Password Submit Button --
        {#if (confirmPassword.length > 0) && showButton}
          <PrimaryButton
            on:click={submitPassword}
            top={24}
            bottom={24}
            standard="Create MinIO"
            success="Setup complete! Toggle Remote Access to view your MinIO Console!"
            failure={submitted ? "An error occured, refresh the page and try again" : "Passwords do not match"}
            loading="Setting up MinIO for you..."
            status={!submitted && (minioPassword == confirmPassword) ? 'standard' : buttonStatus}
          />
        {/if}
  
      {#if nw_label == 'Remote'}

        <!-- Show MinIO Console URL --
        {#if minIO_reg}
          <div class="info">
            <div class="title">MinIO Console</div>

            <div class="login-key-wrapper">
              <div on:click={copyMinIO} id="minio" data-clipboard-text={minIO} class="login-key">
                {
                  clickedMinIO ? "copied!" 
                  : viewMinIO ? minIO
                  : "click to copy"
                }
              </div>
              <a class="newtab" href={minIO} target="_blank">
                <Fa icon={faArrowUpRightFromSquare} size="1.2x" />
              </a>
              <button on:click={()=> viewMinIO = !viewMinIO}>
                <img class="eye" src={viewMinIO ? "/eye-closed.svg" : "/eye-open.svg"} alt="eye" />
              </button>
            </div>
          </div>
        {/if}
      {/if}
    {/if}

    <!-- Toggle Urbit Network -->
		<PierNetwork {name} {remote} {wgReg} {wgRunning} />

<style>
  .setup-title {
    font-weight: 700;
    text-align: left;
  }
  .s3-alert {
    color: orange;
  }
  .title-smaller {
    font-weight: 70;
    margin-bottom: 6px;
    text-align: left;
    font-size: 12px;
  }
  .minio-password {
    font-size: 12px;
    padding: 8px;
    background: #ffffff4d;
    border-radius: 6px;
    flex: 1;
    border: none;
    font-family: inherit;
    color: inherit;
  }
  ::placeholder {
    color: inherit;
    opacity: .6;
  }
  .minio-password:focus {
    outline: none;
  }
  .question-mark {
    color: inherit;
    cursor: pointer;
    margin-left: 4px;
  }
  .alert-mark {
    color: orange;
    cursor: pointer;
    margin-left: 4px;
  }
  .minio-info {
    font-size: 12px;
    margin-bottom: 12px;
    padding-right: 30px;
  }
</style>
