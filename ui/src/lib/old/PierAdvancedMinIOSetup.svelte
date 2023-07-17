<script>
  import { createEventDispatcher } from 'svelte'
  import Fa from 'svelte-fa'
  import { faCheck, faTriangleExclamation } from '@fortawesome/free-solid-svg-icons'
  import { faCircleQuestion } from '@fortawesome/free-regular-svg-icons'

  import { api } from '$lib/api'

  import EyeButton from '$lib/EyeButton.svelte'
  import PrimaryButton from '$lib/PrimaryButton.svelte'

  export let name, minIOReg

  const dispatch = createEventDispatcher()

  let textToggle = 'none'
  let minIOPassword = ''
  let confirmPassword = ''
  let pwdView = false
  let cfmView = false
  let buttonStatus = 'standard'
  let submitted = false
  let showButton = true
  let linkCheck = true


  // Handle info and disclaimer visibility
  const handleTextToggle = val => {
    if (val == textToggle) {
      textToggle = 'none'
    } else { textToggle = val }
  }

  // Toggle first password prompt
  const togglePwdView = () => {
    pwdView = !pwdView
    document.querySelector('#minio-password').type = pwdView ? 'text' : 'password'
  }

  // Toggle confirm password prompt
  const toggleCfmView = () => {
    cfmView = !cfmView
    document.querySelector('#minio-password-1').type = cfmView ? 'text' : 'password'
  }

  // Submit MinIO password
  const submitPassword = () => {
    submitted = true
    buttonStatus = 'loading'

    fetch($api + '/urbit?urbit_id=' + name, {
			method: 'POST',
      credentials: "include",
			headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({
        'app':'minio',
        'password':confirmPassword,
        'link': linkCheck
      })
	  })
      .then(r => r.json())
      .then(d => { 
        if (d == 200) {
          buttonStatus = 'success'
          setTimeout(()=> dispatch('cancel'), 3000)
        }
        else {buttonStatus = 'failure'}
        setTimeout(()=> buttonStatus = 'standard', 3000)
        }
      )
  }

</script>
<div class="main-wrapper">
  <div class="title-wrapper">
    <!-- Request for minIO password if not registered -->
    <div class="title-smaller">MinIO Setup</div>

    <!-- Info button -->
    <button class="question-mark" on:click={()=>handleTextToggle('info')} >
      <Fa icon={faCircleQuestion} size="1.2x" />
    </button>

    <!-- MinIO disclaimer button -->
    <button class="alert-mark" on:click={()=>handleTextToggle('alert')} >
      <Fa icon={faTriangleExclamation} size="1.2x" />
    </button>
  </div>

  <!-- Info text -->
  {#if textToggle == 'info' }
    <div class="minio-info">
       Store and share files on Urbit with MinIO. All data is stored locally on your device.
    </div>
  {/if}

  <!-- MinIO disclaimer text -->
  {#if textToggle == 'alert' }
    <div class="minio-info s3-alert">Warning: if you switch between anchors, it will break your previous S3 links.</div>
  {/if}
          
  <!-- MinIO password length warning -->
  {#if (minIOPassword.length > 0) && (minIOPassword.length < 8)}
    <div class="title-smaller">At least 8 characters</div>
  {:else if (confirmPassword != minIOPassword) && (confirmPassword.length > 0)}
    <div class="title-smaller">Passwords do not match!</div>
  {/if}


  <!-- Password input -->
  <div class="info">
    <div class="title-smaller">Create a Password</div>
    <div class="pier-cred-wrapper">
      <input
        id="minio-password"
        bind:value={minIOPassword}
        class="minio-password"
        type="password"
        placeholder="At least 8 characters" />
      <EyeButton on:click={togglePwdView} view={pwdView} />
    </div>
  </div>

  <!-- Confirm Password -->
  <div class="info">
    <div class="title-smaller">Confirm Password</div>
    <div class="pier-cred-wrapper">
      <input
        id="minio-password-1"
        bind:value={confirmPassword}
        class="minio-password"
        class:pad={confirmPassword.length <= 0}
        type="password"
        placeholder="Enter the password again" />
      <EyeButton on:click={toggleCfmView} view={cfmView} />
    </div>
  </div>

  <!-- Auto link to ship -->
  <div class="link-check" on:click={()=> linkCheck = !linkCheck}>
    <div class="box" class:highlight={linkCheck}>
      {#if linkCheck}
        <Fa icon={faCheck} size="1x"/>
      {/if}
    </div>
    Automatically link to Urbit
  </div>

  <div class="button-wrapper">
    <!-- Password Submit Button -->
    <PrimaryButton
      on:click={()=>dispatch('cancel')}
      standard="Cancel"
      background="#FFFFFF4D"
    />

    <!-- Password Submit Button -->
    <PrimaryButton
      on:click={submitPassword}
      left={false}
      standard="Create MinIO"
      success="Created!"
      failure="An error occured, refresh the page and try again"
      loading="Setting up MinIO for you..."
      status={(minIOPassword == confirmPassword) && (confirmPassword.length >= 8)  ? buttonStatus : 'disabled'}
    />
  </div>
</div>

<style>
  .main-wrapper {
    padding: 0 20px;
  }
  .title-wrapper {
    display: flex;
    margin-bottom: 12px;
  }
  .button-wrapper {
    display: flex;
    margin-top: 12px;
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
    margin-bottom: 12px;
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
  .pad {
    margin-bottom: 16px;
  }
  .minio-password:focus {
    outline: none;
  }
  .question-mark {
    color: inherit;
    cursor: pointer;
  }
  .alert-mark {
    color: orange;
    cursor: pointer;
  }
  .minio-info {
    font-size: 12px;
    margin-bottom: 12px;
    padding-right: 30px;
  }
  .link-check {
    flex: 1;
    display: flex;
    gap: 6px;
    align-items: center;
    justify-content: start;
    text-align: center;
    font-size: 11px;
    cursor: pointer;
    user-select: none;
  }
  .box {
    width: 14px;
    height: 14px;
    background: #ffffff4d;
    border-radius: 4px;
  }
  .highlight {
    background: #028AFB;
  }
</style>
