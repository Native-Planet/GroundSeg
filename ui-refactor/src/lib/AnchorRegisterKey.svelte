<script>
  import { api } from '$lib/api'
  import PrimaryButton from '$lib/PrimaryButton.svelte'

  export let wgReg

  let key = '',
    view = false,
    loading = false,
    buttonStatus = 'standard',
    reRegCheck = true

  const toggleView = () => {
    view = !view
    document.querySelector('#input').type = view ? 'text' : 'password'
  }

  const registerKey = () => {
    buttonStatus = 'loading'
    let module = 'anchor'
	  fetch($api + '/system?module=' + module, {
			method: 'POST',
			headers: {'Content-Type': 'application/json'},
			body: JSON.stringify({'action':'register','key':key.trim()})
	  })
      .then(d => d.json())
      .then(res => {
        if (res === 200) {
          buttonStatus = 'success'
          setTimeout(()=>{buttonStatus = 'standard'; key = ''}, 3000)
        } else {
          console.log(res)
          buttonStatus = 'failure'
          setTimeout(()=> {buttonStatus = 'standard';reRegCheck = true}, 3000)
        }})
      .catch(err => console.log(err))
  }

</script>

<div class="reg-key-wrapper">

  <!-- If not registered -->
  {#if !wgReg}
    <div class="reg-title">Key Registration</div>
    <div class="reg-key">
      <input id='input' type="password" bind:value={key} />
      <img on:click={toggleView} src="/eye-{view ? "closed" : "open"}.svg" alt="eye" />
    </div>

  <!-- if registered -->
  {:else if !reRegCheck}
    <div class="reg-title">Key Registration</div>
    <div class="reg-key">
      <input id='input' type="password" bind:value={key} />
      <img on:click={toggleView} src="/eye-{view ? "closed" : "open"}.svg" alt="eye" />
    </div>
  {/if}

  <!-- Submit button -->
  <PrimaryButton
    on:click={wgReg && reRegCheck ? ()=>reRegCheck = false : registerKey }
    standard="Register{wgReg && reRegCheck ? " A New Key" : ""}"
    success="Key registered"
    failure="Registration failed"
    loading="Processing..."
    status={
      wgReg && reRegCheck ? buttonStatus :
      key == '' ? "disabled" : buttonStatus
    }
    top="{wgReg && reRegCheck ? "26" : "12"}"
  />
</div>

<style>
  .reg-key-wrapper {
    gap: 6px;
    margin-top: 12px;
  }
  .reg-title {
    font-size: 14px;
    padding-bottom: 6px;
  }
  .reg-key {
    display: flex;
  }
  .reg-key > input {
    font-family: inherit;
    background: #ffffff4d;
    color: inherit;
    border-radius: 6px;
    font-size: 12px;
    padding: 8px;
    border: none;
    flex: 1;
  }
  input:focus {
    outline: none;
  }
  .reg-key > img {
    padding-left: 12px;
    opacity: .8;
    cursor: pointer;
  }
  .disabled {
    opacity: .6;
    pointer-events: none;
  }

</style>
