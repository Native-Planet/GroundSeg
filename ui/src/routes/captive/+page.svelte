<script>
  import Fa from 'svelte-fa'
  import { faLock } from '@fortawesome/free-solid-svg-icons';

  import { slide } from 'svelte/transition';
	import { quintOut } from 'svelte/easing';

  let status = ""  // current status
  let selected = "" // selected network to attempt connection
  let pwd = "" // network password

  // reset to home page
  const resetSelected = () => {
    selected = ""
    pwd = ""
    status = ""
  }

  // connect to ssid
  const attemptConnect = () => {
    status = "connecting" 
    setTimeout(fakeFailureState, 10000)
  }

  // debug
  let ssidArr = [
    "Skyline_5GHz",
    "CoffeeBean_Guest",
    "MysticForest",
    "QuantumWave",
    "SilentHill",
    "Hobbiton_Wifi",
    "GalacticZone",
    "NinjaNetwork"
  ];

  const fakeFailureState = () => {
    status = "failed"
    setTimeout(resetSelected, 3000)
  }
  // end debug

</script>
<div class="wrapper">
  <div class="header">
    <div class="c2c-title">Connect to network</div>
    <div class="c2c-subtitle">Choose a wi-fi network</div>
  </div>
  {#if selected.length < 1}
    <div
      class="ssids"
      in:slide={{ delay: 250, duration: 300, easing: quintOut, axis: 'x' }}
      out:slide={{ duration: 300, easing: quintOut, axis: 'x' }}
      >
      {#each ssidArr as ssid}
        <div class="ssid" on:click={()=>selected=ssid}>
          <div class="ssid-name">{ssid}</div>
          <div class="ssid-lock"><Fa icon={faLock} size="1x"/></div>
        </div>
      {/each}
    </div>
  {:else}
    <div class="input-data"
      in:slide={{ delay: 250, duration: 300, easing: quintOut, axis: 'x' }}
      out:slide={{ duration: 300, easing: quintOut, axis: 'x' }}
     >
      <div class="label">Network Name</div>
      <div class="ssid">
        <div class="ssid-name">{selected}</div>
        <div class="ssid-lock"><Fa icon={faLock} size="1x"/></div>
      </div>
      {#if status == "connecting"}
        <div class="status-state"
          in:slide={{ delay: 250, duration: 300, easing: quintOut, axis: 'x' }}
          out:slide={{ duration: 300, easing: quintOut, axis: 'x' }}
          >
          <div class="main-text"
            in:slide={{ delay: 250, duration: 300, easing: quintOut, axis: 'x' }}
            out:slide={{ duration: 300, easing: quintOut, axis: 'x' }}
          >Connection to {selected} has been Requested</div>
          <div class="info-text"
            in:slide={{ delay: 250, duration: 300, easing: quintOut, axis: 'x' }}
            out:slide={{ duration: 300, easing: quintOut, axis: 'x' }}
            >
            Continue your setup on your server’s device. Connection from this device has been disconnected.
          </div>
        </div>
      {:else if status == "failed"}
        <div
          class="status-state"
          in:slide={{ delay: 250, duration: 300, easing: quintOut, axis: 'x' }}
          out:slide={{ duration: 300, easing: quintOut, axis: 'x' }}
          >
          <div 
            class="main-text error"
            in:slide={{ delay: 250, duration: 300, easing: quintOut, axis: 'x' }}
            out:slide={{ duration: 300, easing: quintOut, axis: 'x' }}
               >
               Attempt to connect to {selected} failed!</div>
        </div>
      {:else}
        <div class="label">Network Password</div>
        <input type="password" placeholder="Password" bind:value={pwd} />
        <div class="buttons">
          <button class="back" on:click={resetSelected}>Back</button>
          <button class="connect" on:click={attemptConnect} disabled={pwd.length < 1}>Connect</button>
        </div>
      {/if}
    </div>
  {/if}
</div>

<style>
  .wrapper {
    margin: 91px;
  }
  .header > .c2c-title {
    color: var(--NP_Black, #313933);
    leading-trim: both;
    text-edge: cap;
    font-family: BPdotsUnicase;
    font-size: 48px;
    font-style: normal;
    font-weight: 700;
    line-height: 64px; /* 133.333% */
    letter-spacing: -1.44px;
    text-transform: uppercase;
  }
  .header > .c2c-subtitle {
    color: #000;
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 32px;
    font-style: normal;
    font-weight: 300;
    line-height: 48px; /* 150% */
    letter-spacing: -1.28px;
  }
  .ssids {
    margin-top: 48px;
    display: flex;
    flex-direction: column;
    gap: 16px;
    width: 767px;
  }
  .ssid {
    height: 59px;
    border-radius: 16px;
    border: 2px solid #000;
    display :flex;
    gap: 48px;
  } 
  .ssids > .ssid:hover {
    background: var(--bg-modal);
    cursor: pointer;
  }
  input {
    width: 735px;
    height: 57px;
    border-radius: 16px;
    border: 2px solid #000;
    display :flex;
    gap: 48px;
    outline: none;
    color: #000;
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 32px;
    font-style: normal;
    font-weight: 300;
    letter-spacing: -1.92px;
    padding-left: 24px;
  }
  input::placeholder {
  }
  .ssid-name {
    flex: 1;
    color: #000;
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 32px;
    font-style: normal;
    font-weight: 300;
    letter-spacing: -1.92px;
    margin: auto;
    padding-left: 24px;
  }
  .ssid-lock {
    color: #000;
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 32px;
    font-style: normal;
    font-weight: 300;
    letter-spacing: -1.92px;
    margin: auto;
    padding-right: 24px;
  }
  .input-data {
    margin-top: 48px;
    width: 767px;
  }
  .input-data > .ssid {
    margin-bottom: 32px;
  }
  .label {
    margin-bottom: 16px;
    color: var(--Gray-400, #5C7060);
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 32px;
    font-style: normal;
    font-weight: 300;
    letter-spacing: -1.92px;
  }
  .buttons {
    margin-top: 48px;
    display: flex;
    gap: 16px;
  }
  button {
    color: #FFF;
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 32px;
    font-style: normal;
    font-weight: 300;
    line-height: 24px; /* 75% */
    letter-spacing: -1.92px;
    border-radius: 16px;
    padding: 0 48px;
    height: 65px;
    cursor: pointer;
  }
  button:disabled {
    opacity: .6;
    pointer-events: none;
  }
  .back {
    background: var(--Gray-400, #5C7060);
  }
  .connect {
    border-radius: 16px;
    background: #08A317;
  }
  .status-state {
    margin-top: 48px;
    border-radius: 8px;
    background: var(--NP_Gray, #E8E8E3);
    display: flex;
    width: calc(767px - 64px);
    padding: 32px;
    flex-direction: column;
    align-items: flex-start;
    gap: 32px;
  }
  .main-text {
    color: #000;
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 32px;
    font-style: normal;
    font-weight: 300;
    line-height: 48px; /* 150% */
    letter-spacing: -1.92px;
  }
  .info-text {
    margin-top: 32px;
    color: #000;
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    line-height: 32px; /* 133.333% */
    letter-spacing: -1.44px;
  }
  @keyframes breathe {
    0% {
      opacity: 0.1;
    }
    40% {
      opacity: 1;
    }
    60% {
      opacity: 1;
    }
    100% {
      opacity: 0.1;
    }
  }
  .error {
    color: red;
    animation: none;
  }
</style>
