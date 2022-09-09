<script>
  import { api } from '$lib/api'
  import { onMount, onDestroy } from 'svelte'
  import Select from 'svelte-select'
  import PrimaryButton from '$lib/PrimaryButton.svelte'

  export let info

  let networks,
    opened = true,
    refreshing = false,
    buttonStatus = 'standard'

  let ethSwapping = false,
    nw = '', pw = '',
    view = false

  onMount(()=> {getNetworks()})
  onDestroy(()=> {opened = false})


  const toggleEth = () =>  {

    ethSwapping = true

    let u = api + "/settings/eth-only"
    const f = new FormData()
    f.append('ethernet', !info.ethOnly)

    fetch(u, {method: 'POST',body: f})
      .then(r => r.json())
      .then(d => { if (d == 200) {
        ethSwapping = false
        console.log("swapped")
   }})}

  const connectToNetwork = () =>  {

    buttonStatus = 'loading'

    let u = api + "/settings/connect"
    const f = new FormData()
    f.append('network', nw)
    f.append('password', pw)

    fetch(u, {method: 'POST',body: f})
      .then(r => r.json())
      .then(d => { if (d == 200) {
        buttonStatus = 'success'
        setTimeout(()=>{
          buttonStatus = 'standard'
          nw = info.connected
        }, 3000)
   }})}

  const toggleView = () => {
    view = !view
    document.querySelector('#pass').type = view ? 'text' : 'password'
  }

  const getNetworks = () => {
    if (opened) {
      fetch(api + "/settings/networks").then(r => r.json()).then(d => networks = d)
      setTimeout(getNetworks, 10000)
   }}

</script>

{#if info}
  <div class="network">
    <div class="network-title">Connectivity</div>
    <div class="ethernet">
      <div class="ethernet-text" class:disabled={!info.ethOnly}>Ethernet Only</div>
      <div on:click={toggleEth} class="switch-wrapper">
        <div class="switch {info.ethOnly ? "on" : "off"}"></div>
      </div>
    </div>

    {#if !info.ethOnly}
      <div class="wifi">
        <div class="select">
          <Select
            items={networks}
            listPlacement="auto"
            placeholder="Select Network"
            value={nw == '' ? info.connected : nw}
            on:clear={()=> nw = ''}
            on:select={e => nw = e.detail.value} />
        </div>

        {#if (info.connected !== nw) && (nw.length > 0)}
          <div class="wifi-pass-wrapper">
            <div class="pass-text">Wifi Password</div>
            <div class="wifi-pass">
              <input id='pass' type="password" bind:value={pw} />
              <img on:click={toggleView} src="/eye-{view ? "closed" : "open"}.svg" alt="eye" />
            </div>
            <PrimaryButton
              on:click={connectToNetwork}
              standard="Connect"
              success="Connected to network"
              failure="Connection failed"
              loading="Connecting"
              status={(pw == '') || (nw == null)? "disabled" : buttonStatus}
              top="12" />
          </div>
        {/if}
      </div>
    {/if}
  </div>
{:else}
  <div class="network">
    <div class="network-title">Connectivity</div>
    <div class="ethernet">
      <div class="ethernet-text" class:disabled={true}>Ethernet Only</div>
      <div class="switch-wrapper-blurred"></div>
    </div>
    <div class="blurred"></div>


  </div>
{/if}

<style>
@keyframes breathe {
    0% {opacity: .6}
    50% {opacity: 0}
    100% {opacity: .6}
  }
  .network {
    background: #0000006d;
    width: 300px;
    padding: 40px;
    border-radius: 15px;
    font-size: 18px;
    gap: 12px;
  }
  .network-title {
    font-size: 18px;
    padding-bottom: 8px;
  }
  .ethernet {
    display: flex;
    padding-top: 12px;
  }
  .ethernet-text {
    font-size: 14px;
    flex: 1;
  }
  .switch-wrapper {
    border-radius: 8px;
    width: 32px;
    height: 12px;
    background: #ffffff4d;
    padding: 2px;
  }
  .switch-wrapper-blurred {
    border-radius: 8px;
    width: 32px;
    height: 12px;
    background: #ffffff4d;
    padding: 2px;
    filter: blur(10px);
    animation: breathe 2s infinite;
  }
  .blurred {
    height: 32px;
    width: 100%;
    background: #ffffff4d;
    border-radius: 8px;
    margin-top: 20px;
    filter: blur(10px);
    animation: breathe 2s infinite;
  }
  .switch {
    height: 100%;
    width: 19px;
    border-radius: 6px;
    margin-top: auto;
    cursor: pointer;
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
  .wifi {
    margin-top: 20px;
  }
  .select {
    --background: #ffffff4d;
    --border: none;
    --borderRadius: 8px;
    --inputColor: #ffffff;
    --inputPadding: 12px;
    --listBackground: #3d3d3d;
    --itemHoverBG: #0000004d;
    --itemIsActiveBG: #000;
    --placeholderColor: #fff;
    --height: 32px;
    font-size: 12px;
    font-weight: 700;
    border-radius: 8px;
  }
  .wifi-pass-wrapper {
    display: flex;
    flex-direction: column;
    gap: 6px;
    margin-top: 12px;
  }
  .pass-text {
    font-size: 12px;
  }
  .wifi-pass {
    display: flex;
  }
  .wifi-pass > input {
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
  .wifi-pass > img {
    padding-left: 12px;
    opacity: .8;
    cursor: pointer;
  }
  .disabled {
    opacity: .6;
    pointer-events: none;
  }

</style>
