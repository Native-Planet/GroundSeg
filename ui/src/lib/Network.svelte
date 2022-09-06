<script>
  import { api } from '$lib/api'
  import Select from 'svelte-select'

  export let info

  let ethSwapping = false,
    nw = null, pw = '',
    view = false

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

  const toggleView = () => {
    view = !view
    document.querySelector('#pass').type = view ? 'text' : 'password'
  }

  // placeholder
  let ssid = ["John's Wifi","Native Planet 5G","City Wok"]

</script>

  {JSON.stringify(info)}
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
            items={info.networks}
            listPlacement="auto"
            placeholder="Select Network"
            on:clear={()=> nw = null}
            on:select={e => nw = e.detail.value} />
        </div>

        {#if info.connected !== nw}
          <div class="wifi-pass-wrapper">
            <div class="pass-text">Wifi Password</div>
            <div class="wifi-pass">
              <input id='pass' type="password" bind:value={pw} />
              <img on:click={toggleView} src="/eye-{view ? "closed" : "open"}.svg" alt="eye" />
            </div>
            <button class="connect" class:disabled={(pw == '') || (nw == null)}>Connect</button>
          </div>
        {/if}
      </div>
    {/if}
  </div>
{/if}

<style>
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
  .connect {
    background: #008eff;
    padding: 8px;
    color: inherit;
    border: none;
    border-radius: 6px;
    font-family: inherit;
    width: 80px;
    margin-top: 6px;
    cursor: pointer;
  }
  .disabled {
    opacity: .6;
    pointer-events: none;
  }

</style>
