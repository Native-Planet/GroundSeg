<script>
  import { api } from '$lib/api'
  import PrimaryButton from '$lib/PrimaryButton.svelte'

  export let info

  let key = '',
    view = false,
    loading = false,
    buttonStatus = 'standard'

  const toggleView = () => {
    view = !view
    document.querySelector('#input').type = view ? 'text' : 'password'
  }

  const toggleAnchor = () => {
    loading = true
    const f = new FormData()
    const u = api + "/settings/anchor"
    f.append('anchor', !info.anchor)
    fetch(u, {method: 'POST',body: f})
      .then(d => d.json())
      .then(res => {
        if (res === 200) {
          loading = false
    }})}

  const registerKey = () => {
    buttonStatus = 'loading'
    const f = new FormData()
    const u = api + "/settings/anchor/register"
    f.append('key', key)
    fetch(u, {method: 'POST',body: f})
      .then(d => d.json())
      .then(res => {
        if (res === 200) {
          buttonStatus = 'success'
          setTimeout(()=>{
            buttonStatus = 'standard'
            key = ''
          }, 3000)
        }
        if (res === 400) {
          buttonStatus = 'failure'
          setTimeout(buttonStatus = 'standard', 3000)
        }
      })
   }

</script>

<div class="anchor">
  <div class="title-wrapper">
    <div class="title">Anchor</div>
    {#if info}
      <div
        on:click={toggleAnchor}
        class:disabled={loading}
        class="switch-wrapper">
        <div class="switch {info.anchor ? "on" : "off"}"></div>
      </div>
    {:else}
      <div class="blurred"><br></div>
    {/if}
  </div>
  <div class="reg-key-wrapper">
    <div class="reg-title">Key Registration</div>
    <div class="reg-key">
      <input id='input' type="password" bind:value={key} />
      <img on:click={toggleView} src="/eye-{view ? "closed" : "open"}.svg" alt="eye" />
    </div>
    <PrimaryButton
        on:click={registerKey}
        standard="Register"
        success="Key registered"
        failure="Registration failed"
        loading="Processing..."
        status={key == '' ? "disabled" : buttonStatus}
        top="12" />
  </div>
</div>

<style>
  @keyframes breathe {
    0% {opacity: .6}
    50% {opacity: 0}
    100% {opacity: .6}
  }
  .anchor {
    background: #0000006d;
    width: 300px;
    padding: 40px;
    border-radius: 15px;
    font-size: 18px;
  }
  .title-wrapper {
    display: flex;
    align-items: center;
  }
  .title {
    font-size: 18px;
    flex: 1;
  }
  .switch-wrapper {
    border-radius: 8px;
    width: 32px;
    height: 12px;
    background: #ffffff4d;
    padding: 2px;
    cursor: pointer;
  }
  .blurred {
    width: 36px;
    animation: breathe 2s infinite;
    background: #ffffff4d;
    filter: blur(6px);
    border-radius: 8px;
  }
  .switch {
    height: 100%;
    width: 19px;
    border-radius: 6px;
    margin-top: auto;
  }
  .on {
    background: #008eff;
    float: right;
  }
  .off {
    background: #000;
    float: left;
    opacity: .2;
  }
  .reg-key-wrapper {
    gap: 6px;
    margin-top: 12px;
  }
  .reg-title {
    font-size: 14px;
    padding-bottom: 4px;
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
