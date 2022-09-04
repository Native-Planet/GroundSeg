<script>
  import Select from 'svelte-select'

  let ethOnly = false, nw = null, pw = '', view = false

  const toggleEth = () => ethOnly = !ethOnly

  const toggleView = () => {
    view = !view
    document.querySelector('#pass').type = view ? 'text' : 'password'
  }

  // placeholder
  let ssid = ["John's Wifi","Native Planet 5G","City Wok"]

</script>

  <div class="network">
    <div class="network-title">Connectivity</div>
    <div class="ethernet">
      <div class="ethernet-text" class:disabled={!ethOnly}>Ethernet Only</div>
      <div on:click={toggleEth} class="switch-wrapper">
        <div class="switch {ethOnly ? "on" : "off"}"></div>
      </div>
    </div>

    {#if !ethOnly}
      <div class="wifi">
        <div class="select">
          <Select
            items={ssid}
            listPlacement="auto"
            placeholder="Select Network"
            on:clear={()=> nw = null}
            on:select={e => nw = e.detail.value} />
        </div>

        <div class="wifi-pass-wrapper">
          <div class="pass-text">Wifi Password</div>
          <div class="wifi-pass">
            <input id='pass' type="password" bind:value={pw} />
            <img on:click={toggleView} src="/eye-{view ? "closed" : "open"}.svg" alt="eye" />
          </div>
          <button class="connect" class:disabled={(pw == '') || (nw == null)}>Connect</button>
        </div>
      </div>
    {/if}
  </div>

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
  }
  .disabled {
    opacity: .6;
    pointer-events: none;
  }

</style>
