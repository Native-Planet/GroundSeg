<script>
  import { page } from '$app/stores'
  import { power, api } from '$lib/api'
  import Fa from 'svelte-fa'
  import { faPowerOff, faRotateRight } from '@fortawesome/free-solid-svg-icons'

  const shutdown = () => {
    console.log("shutdown")
  }
    /*
    let u = $api + "/settings/shutdown"
    const f = new FormData()

    setTimeout(()=> window.location.href = "/", 3000)

    fetch(u, {method: 'POST',body: f})
      .then(r => r.json())
      .then(d => { if (d == 200) {
        window.location.href = "/"
   }})}
    */
  const restart = () => {
    console.log("restart")
  }
    /*
    let u = $api + "/settings/restart"
    const f = new FormData()

    fetch(u, {method: 'POST',body: f})
      .then(r => r.json())
      .then(d => { if (d == 200) {
        window.location.href = "/"
   }})}
  */

  const cancel  = () => {
    power.set(null)
  }
</script>

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
<style>
  @font-face {
    font-family: Inter;
    src: url("/Inter-SemiBold.otf");
  }
  .power {
    font-family: Inter;
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
</style>
