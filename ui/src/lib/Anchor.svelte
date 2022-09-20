<script>
  import { api } from '$lib/api'
  import PrimaryButton from '$lib/PrimaryButton.svelte'
  import Fa from 'svelte-fa'
  import { faChevronDown, faChevronUp } from '@fortawesome/free-solid-svg-icons/index.es'


  export let info

  let key = '',
    view = false,
    loading = false,
    buttonStatus = 'standard',
    reRegCheck = true,
    advanced = false,
    epKey = '',
    curEpKey = '',
    defaultEpKey = 'api.startram.io',
    epButtonStatus = 'standard'

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
    f.append('key', key.trim())
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
          setTimeout(()=>buttonStatus = 'standard', 3000)
  }})}

  const insertNP = () => epKey = defaultEpKey

  const toggleAdvanced = () => {
    if (!advanced) {getCurrentEndpoint()}
    advanced = !advanced
  }
  
  const getCurrentEndpoint = () =>  {
    const u = api + "/settings/anchor/endpoint"
    fetch(u).then(d=>d.json()).then(r=>{epKey=r;curEpKey=r})
  }

  const connectEndpoint = () => {
    epButtonStatus = 'loading'
    const u = api + "/settings/anchor/endpoint"
    const f = new FormData()
    f.append('new',epKey.trim())
    fetch(u,{method:'POST',body:f})
      .then(d=>d.json()).then(r=>{
        if (r === 200) {
          epButtonStatus = 'success'
          setTimeout(()=>{
            epButtonStatus = 'standard'
            getCurrentEndpoint()
          }, 3000)}
        if (r === 400) {
          epButtonStatus = 'failure'
          setTimeout(()=>epButtonStatus = 'standard', 3000)
       
   }})}

</script>

<div class="anchor">

{#if info}

  <!-- title and toggle button -->
  <div class="title-wrapper">
    <div class="title">Anchor</div>
    {#if info.wg_reg}
      <div
        on:click={toggleAnchor}
        class:disabled={loading}
        class="switch-wrapper">
        <div class="switch {info.anchor ? "on" : "off"}"></div>
      </div>
    {/if}
  </div>

  <div class="reg-key-wrapper">
    {#if !info.wg_reg}
      <div class="reg-title">Key Registration</div>
      <div class="reg-key">
        <input id='input' type="password" bind:value={key} />
        <img on:click={toggleView} src="/eye-{view ? "closed" : "open"}.svg" alt="eye" />
      </div>
    {:else}
      {#if !reRegCheck}
      <div class="reg-title">Key Registration</div>
      <div class="reg-key">
        <input id='input' type="password" bind:value={key} />
        <img on:click={toggleView} src="/eye-{view ? "closed" : "open"}.svg" alt="eye" />
      </div>
      {/if}
    {/if}
    <PrimaryButton
      on:click={info.wg_reg && reRegCheck ? ()=>reRegCheck = false : registerKey }
      standard="Register{info.wg_reg && reRegCheck ? " A New Key" : ""}"
      success="Key registered"
      failure="Registration failed"
      loading="Processing..."
      status={
        info.wg_reg && reRegCheck ? buttonStatus :
        key == '' ? "disabled" : buttonStatus
      }
      top="{info.wg_reg && reRegCheck ? "26" : "12"}"
    />
  </div>

  <div class="reg-key-wrapper">
    <div class="advanced" on:click={toggleAdvanced}>
      Advanced Options
      <Fa icon={advanced ? faChevronUp : faChevronDown} size="0.8x" />
    </div>

    {#if advanced}
      <div class="ep-title">Set Endpoint</div>
      <div class="ep-key">
        <input type="text" bind:value={epKey} />
        <img on:click={insertNP} width="24px" src="/nplogo.svg" alt="np logo" />
      </div>

      {#if curEpKey != epKey}
        <PrimaryButton
          on:click={connectEndpoint}
          standard="Set to {defaultEpKey == epKey ? "Native Planet" : "Custom"} Endpoint"
          success="Endpoint successfully changed"
          failure="Failed to change endpoint"
          loading="Connecting to your new endpoint.."
          status={epButtonStatus}
          top="12"
        />
      {/if}
    {/if}
  </div>


{:else}

  <div class="title-wrapper">
    <div class="title">Anchor</div>
    <div class="blurred"><br></div>
  </div>
  <div class="blurred-block"></div>

{/if}
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
  .blurred-block {
    animation: breathe 2s infinite;
    width: 80%;
    padding-left: 10%;
    padding-right: 10%;
    background: #ffffff4d;
    margin-top: 26px;
    border-radius: 8px;
    height: 30px;
    filter: blur(10px);

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
  .advanced {
    font-size: 14px;
    padding-top: 6px;
    cursor: pointer;
    width: 150px;
  }
  .advanced:hover {
    opacity: .6;
  }

  .ep-title {
    margin-top: 18px;
    font-size: 14px;
    padding-bottom: 6px;
  }
  .ep-key {
    display: flex;
  }
  .ep-key > input {
    font-family: inherit;
    background: #ffffff4d;
    color: inherit;
    border-radius: 6px;
    font-size: 12px;
    padding: 8px;
    border: none;
    flex: 1;
  }
  .ep-key > img {
    padding-left: 12px;
    opacity: .8;
    cursor: pointer;
  }

</style>
