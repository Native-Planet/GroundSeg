<script>
  import { onMount } from 'svelte'
  import { layout } from '$lib/components'
  import { page } from '$app/stores'
  import { power, api } from '$lib/api'
  import Fa from 'svelte-fa'
  import { faPowerOff, faRotateRight, faDownload } from '@fortawesome/free-solid-svg-icons/index.es'

  let hasUpdate = false

  const shutdown = () => {
    let u = api + "/settings/shutdown"
    const f = new FormData()

    setTimeout(()=> window.location.href = "/", 3000)

    fetch(u, {method: 'POST',body: f})
      .then(r => r.json())
      .then(d => { if (d == 200) {
        window.location.href = "/"
   }})}

  const restart = () => {
    let u = api + "/settings/restart"
    const f = new FormData()

    fetch(u, {method: 'POST',body: f})
      .then(r => r.json())
      .then(d => { if (d == 200) {
        window.location.href = "/"
   }})}
  const cancel  = () => {
    power.set(null)
  }

  const update = () => {
    let u = api + "/settings/update"
    fetch(u).then(r => r.json()).then(d => hasUpdate = d)
    setTimeout(update, (15 * 60 * 1000))}

  const downloadUpdate = () => {
    let u = api + "/settings/update"
    const f = new FormData()
    fetch(u, {method: 'POST',body: f})
      .then(window.location.href = "/")}
//      .then(r => r.json())
//      .then(d => { if (d == 200) {
//        window.location.href = "/"
//   }})}

  onMount(()=> update())

</script>

<div class="container" class:frozen={($page.url.pathname === "/settings") && (($power === 'shutdown') || ($power === 'restart'))}>
  <div class='slot'>
      <slot />
  </div>
</div>
{#if $page.url.pathname === "/settings"}
  {#if $power === 'shutdown'}
    <div class="power">
      <div class="text">Are you sure you want to shut down the device?</div>
      <div class="buttons">
        <button class="cancel" on:click={cancel}>Cancel</button>
        <button class="shutdown" on:click={shutdown}>
          <Fa icon={faPowerOff} size="0.85x" />
          <span>Shutdown</span>
        </button>
      </div>
    </div>
  {:else if $power === 'restart'}
    <div class="power">
      <div class="text">Are you sure you want to restart the device?</div>
      <div class="buttons">
        <button class="cancel" on:click={cancel}>Cancel</button>
        <button class="restart" on:click={restart}>
          <Fa icon={faRotateRight} size="0.85x" />
          <span>Restart</span>
        </button>
      </div>
    </div>
  {/if}
{/if}
{#if !($page.url.pathname === "/settings")}
  <svelte:component this={layout.settings} />
{/if}

{#if hasUpdate}
  <button on:click={downloadUpdate} class="has-update"><Fa icon={faDownload} size="2x" /><span>Download new update!</span></button>
{/if}

<style>

  .power {
    width: 100vw;
    transform: translate(-50%, -50%);
    background: #0404044d;
    position: absolute;
    top: 50%;
    left: 50%;
    text-align: center;
    backdrop-filter: blur(10px);
    -moz-backdrop-filter: blur(10px);
    -o-backdrop-filter: blur(10px);
    -webkit-backdrop-filter: blur(10px);
  }
  .text {
    color: #ffffff;
    font-family: inherit;
    padding: 60px;
  }
  .buttons {
    width: 300px;
    margin: auto;
    margin-bottom: 60px;
    display: flex;
  }
  .buttons > button {
    padding: 6px 12px 6px 12px;
    border: none;
    font-size: 14px;
    cursor: pointer;
  }
  span {
    padding-left: 4px;
  }
  .shutdown {
    color: red;
    background: none;
    margin-left: auto;
    font-weight: 500;
  }
  .restart {
    color: orange;
    background: none;
    margin-left: auto;
    font-weight: 700;
  }
  .cancel {
    color: white;
    background: #ffffff4d;
    border-radius: 6px;
    margin-right: auto;
  }
  .container {
    position: fixed;
    top: 50%;
    left: 50%;
    transform: translate(-50%, -50%);
    color: #fff;
    max-height: 80vh;
    -ms-overflow-style: none;
    scrollbar-width: none;
  }
  .container::-webkit-scrollbar {
    display: none;
  }
  .slot {
    background: rgba(109, 109, 109, 0.5);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 20px;
    backdrop-filter: blur(60px);
    -moz-backdrop-filter: blur(60px);
    -o-backdrop-filter: blur(60px);
    -webkit-backdrop-filter: blur(60px);
  }
  .frozen {
    opacity: 0;
    pointer-events: none;
  }

  .has-update {
    color: cyan;
    font-size: 12px;
    position: absolute;
    bottom: 20px;
    right: 20px;
    padding: 8px;
    border-radius: 8px;
    background: none;
    outline: none;
    border: solid 1px cyan;
    display: flex;
    gap: 6px;
    align-items: center;
    cursor: pointer;
  }
</style>
