<script>
  import { api } from '$lib/api'
  import PrimaryButton from '$lib/PrimaryButton.svelte'
  import Clipboard from 'clipboard'
  import Fa from 'svelte-fa'
  import { faArrowUpRightFromSquare } from '@fortawesome/free-solid-svg-icons/index.es'
  import { faCircleQuestion } from '@fortawesome/free-regular-svg-icons'

  export let name, nw_label, code, ext, minIO, wg_reg, wg_running, minIO_reg

  let viewLogin = false, clickedLogin = false,
    viewExt = false, clickedExt = false,
    viewMinIO = false, clickedMinIO = false,
    isSwitching = false, minioPassword = '', viewKey = false,
    viewConfirm = false, confirmPassword = '' , showMinIOInfo = false,
    buttonStatus = 'failure', submitted = false, showButton = true

  // Toggle Password Visibility
  const toggleViewKey = () => {
    viewKey = !viewKey
    document.querySelector('#minio-password').type = viewKey ? 'text' : 'password'
  }

  const toggleViewConfirm = () => {
    viewConfirm = !viewConfirm
    document.querySelector('#minio-password-1').type = viewConfirm ? 'text' : 'password'
  }


  // Submit MinIO pasword
  const submitPassword = () => {
    let u = api + "/urbit/minio/register"
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
  let copyExt = new Clipboard('#ext');
  let copyMinIO = new Clipboard('#minio');

  copyLogin.on("success", ()=> {
    clickedLogin = true; setTimeout(()=> clickedLogin = false, 1000)})

  copyExt.on("success", ()=> {
    clickedExt = true; setTimeout(()=> clickedExt = false, 1000)})

  copyMinIO.on("success", ()=> {
    clickedMinIO = true; setTimeout(()=> clickedMinIO = false, 1000)})

  // Network switching

  const toggleNetwork = () => { 
    isSwitching = true
    let u = api + "/urbit/network"
    const f = new FormData()
    f.append(name,'network')

    fetch(u, {method: 'POST',body: f})
      .then(r => r.json())
      .then(d => { if (d == 200) {
        isSwitching = false
   }})}


</script>
    
    {buttonStatus}
    <!-- Landscape +code -->
    <div class="info">
      <div class="title">Login Key</div>
      <div class="login-key-wrapper">
        <div on:click={copyLogin} id="login" data-clipboard-text={code} class="login-key">
          {
            clickedLogin ? "copied!" 
            : viewLogin ? code
            : "click to copy"
          }
        </div>
        <button on:click={()=> viewLogin = !viewLogin}>
          <img class="eye" src={viewLogin ? "/eye-closed.svg" : "/eye-open.svg"} alt="eye" />
        </button>
      </div>
    </div>

    <!-- Landscape URL -->
    <div class="info">
      <div class="title">External Access URL</div>
      <div class="login-key-wrapper">
        <div on:click={copyExt} id="ext" data-clipboard-text={ext} class="login-key">
          {
            clickedExt ? "copied!" 
            : viewExt ? ext
            : "click to copy"
          }
        </div>
        <a class="newtab" href={ext} target="_blank">
          <Fa icon={faArrowUpRightFromSquare} size="1.2x" />
        </a>
        <button on:click={()=> viewExt = !viewExt}>
          <img class="eye" src={viewExt ? "/eye-closed.svg" : "/eye-open.svg"} alt="eye" />
        </button>
      </div>
    </div>

    {#if wg_reg && wg_running}

      <!-- Request for minIO password if not registered -->
      {#if !minIO_reg}
        <div class="info">
          <div class="title">
            <span>Setup MinIO Password</span>
            <button class="question-mark" on:click={()=> showMinIOInfo = !showMinIOInfo} >
              <Fa icon={faCircleQuestion} size="1.2x" />
            </button>
          </div>

          {#if showMinIOInfo}
            <div class="minio-info">
              Store and share files on Urbit with MinIO. All data is stored locally on your device.
            </div>
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

      <!-- Confirm Password -->
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

      <!-- Password Submit Button -->
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

      <!-- Show MinIO Console URL -->
      {#if minIO_reg && (nw_label == 'Remote')}
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

    <!-- Toggle Network -->
    {#if wg_reg && wg_running}
    <div class="info"class:switching={isSwitching} on:click={toggleNetwork}>
      <div class="title">Access</div>
      <div class="access-options">
        <button class="option" class:access-active={nw_label === 'Local'} >Local</button>
        <button class="option" class:access-active={nw_label === 'Remote'} >Remote</button>
      </div>
    </div>
  {/if}

<style>
  button {
    appearance: none;
    background: none;
    border: none;
    padding: 0;
    margin: 0;
    height: 32px;
  }
  .info {
    margin-bottom: 12px;
  }
  .title {
    font-weight: 700;
    margin-bottom: 12px;
    text-align: left;
  }
  .title-smaller {
    font-weight: 70;
    margin-bottom: 6px;
    text-align: left;
    font-size: 12px;

  }
  .login-key-wrapper {
    display: flex;
  }
  .login-key {
    font-style: italic;
    font-size: 12px;
    padding: 8px;
    background: #ffffff4d;
    border-radius: 6px;
    flex: 1;
    cursor: pointer;
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
  .minio-info {
    font-size: 12px;
    margin-bottom: 24px;
  }
  .eye {
    height: 32px;
    opacity: .8;
    margin-left: 12px;
    cursor: pointer;
  }
  .newtab {
    margin: auto;
    margin-left: 16px;
    opacity: .8;
  }
  .access-options {
    display: flex;
    width: 240px;
    border-radius: 8px;
    background: #ffffff4d;
    gap: 2px;
  }
  .option {
    color: inherit;
    font-size: 14px;
    flex: 1;
    padding: 8px 0 8px 0;
    background: none;
    border-radius: 8px;
    border: none;
    font-weight: 700;
    cursor: pointer;
  }
  .switching {
    opacity: .6;
    pointer-events: none;
  }
  .access-active {
    background: #008eff;
  }

</style>
