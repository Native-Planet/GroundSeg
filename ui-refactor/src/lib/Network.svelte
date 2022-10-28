<script>
  import { api } from '$lib/api'
  import PrimaryButton from '$lib/PrimaryButton.svelte'

  import NetworkEth from '$lib/NetworkEth.svelte'

  export let ethOnly

  let networks,
    opened = true,
    refreshing = false,
    buttonStatus = 'standard',
    ethSwapping = false,
    nw = '', pw = '',
    view = false

  const toggleEth = () =>  {

    ethSwapping = true

    let u = $api + "/settings/eth-only"
    const f = new FormData()
    f.append('ethernet', !info.ethOnly)

    fetch(u, {method: 'POST',body: f})
      .then(r => r.json())
      .then(d => { if (d == 200) {
        ethSwapping = false
   }})}

  const connectToNetwork = () =>  {

    buttonStatus = 'loading'

    let u = $api + "/settings/connect"
    const f = new FormData()
    f.append('network', nw)
    f.append('password', pw)
    f.append('connected', info.connected)

    fetch(u, {method: 'POST',body: f})
      .then(r => r.json())
      .then(d => { if (d == 200) {
        buttonStatus = 'success'
        setTimeout(()=>{
          buttonStatus = 'standard'
          nw = info.connected
          pw = ''
        }, 3000)
      } else {
        buttonStatus = 'failure'
        setTimeout(()=>{
          buttonStatus = 'standard'
        }, 3000)
      }})}

  const toggleView = () => {
    view = !view
    document.querySelector('#pass').type = view ? 'text' : 'password'
  }

</script>

  <div class="network">
    <div class="network-title">Connectivity</div>
    <NetworkEth {ethOnly} />

    {#if !ethOnly}
      show wifi
    <!--
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
    -->
    {/if}

  </div>

<style>
  .network {
    background: #0404044d;
    padding: 40px;
    border-radius: 15px;
    font-size: 18px;
    gap: 12px;
  }
  .network-title {
    font-size: 18px;
    padding-bottom: 8px;
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
