<script>
  import { api } from '$lib/api'
  import { createEventDispatcher } from 'svelte'
  import PrimaryButton from '$lib/PrimaryButton.svelte'
  import EyeButton from '$lib/EyeButton.svelte'

  let curPass = '', newPass = '', confirmPass = ''
  let curView = false, newView = false, cfmView = false
  let buttonStatus = 'standard'

  const dispatch = createEventDispatcher()
  const cancelMod = () => dispatch('cancel')

  const toggleCur = () => {
    curView = !curView
    document.querySelector('#cur').type = curView ? 'text' : 'password'
  }
  const toggleNew = () => {
    newView = !newView
    document.querySelector('#new').type = newView ? 'text' : 'password'
  }
  const toggleCfm = () => {
    cfmView = !cfmView
    document.querySelector('#cfm').type = cfmView ? 'text' : 'password'
  }

  const submitNewPass = () => {
    buttonStatus = 'loading'
    let module = 'session'
	  fetch($api + '/system?module=' + module, {
			method: 'POST',
      credentials: 'include',
			headers: {'Content-Type': 'application/json'},
			body: JSON.stringify({'action':'change-pass','old-pass':curPass.trim(),'new-pass':confirmPass.trim()})
	  })
      .then(d => d.json())
      .then(res => {
        if (res === 200) {
          buttonStatus = 'success'
          setTimeout(()=> {
            buttonStatus = 'standard'
            dispatch('cancel')
          }, 3000)
        } else {
          buttonStatus = 'failure'
          setTimeout(()=> buttonStatus = 'standard', 3000)
        }
      })
      .catch(err => {
        console.log(err)
        buttonStatus = 'failure'
        setTimeout(()=> buttonStatus = 'standard', 3000)
      })
  }

</script>

<div class="input-wrapper">

  <div class="pwd-wrapper">
    <input id="cur" type="password" placeholder="Current Password" bind:value={curPass} />
    <EyeButton on:click={toggleCur} view={curView} />
  </div>

  {#if curPass.length > 0}
  <div class="pwd-wrapper">
    <input id="new" type="password" placeholder="New Password" bind:value={newPass} />
    <EyeButton on:click={toggleNew} view={newView} />
  </div>
  {/if}

  {#if newPass.length > 0}
    <div class="pwd-wrapper">
      <input id="cfm" type="password" placeholder="Confirm New Password" bind:value={confirmPass} />
      <EyeButton on:click={toggleCfm} view={cfmView} />
    </div>
  {/if}

</div>

<div class="buttons-wrapper">
  <PrimaryButton
    top=12
    background="#ffffff4d"
    standard="Cancel"
    on:click={cancelMod}
  />

  {#if (newPass == confirmPass) && confirmPass.length > 0}
    <PrimaryButton
      top=12
      left={false}
      status={buttonStatus}
      standard="Submit New Password"
      loading="Changing password..."
      success="Password changed!"
      failure="Failed to change password"
      on:click={submitNewPass}
    />
  {/if}
</div>

<style>
  .input-wrapper {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }
  .pwd-wrapper {
    display: flex;
  }
  .buttons-wrapper {
    display: flex;
  }
  input {
    font-size: 12px;
    padding: 8px;
    background: #ffffff4d;
    border-radius: 6px;
    flex: 1;
    border: none;
    font-family: inherit;
    color: inherit;
  }
  input:focus {
    outline: none;
  }
  ::placeholder {
    color: inherit;
    opacity: .6;
  }
</style>
