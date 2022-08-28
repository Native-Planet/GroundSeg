<script>
  import { onMount } from 'svelte'
  import { url } from '/src/Scripts/server'
  import { page } from '$app/stores';
  import Logo from '/src/Components/Buttons/Logo.svelte'

  const cur = $page.url;
  const path = cur.pathname.replace("/", "")
  let pier, access, key

  onMount(async () => {
    const u = url + "/urbit/pier?pier=" + path
    fetch(u).then(r => r.json()).then(d => pier = d)
  })

</script>

<Logo />
<div class="ship">
  {#if pier}
    <div class="card">
      <div class="sigil"></div>
      <div class="info">
        <div class="status {pier.running ? "running" : ""}">
          {pier.running ? "Running" : "Stopped"} 
        </div>
        <div class="patp">
          {pier.name}
          <span class="nick">Nickname</span>
        </div>
      </div>
    </div>
    <div class="info">
      <div class="title">Login Key</div>
      <input spellcheck="false" type="password" bind:value={key}/>
    </div>
    <div class="info">
      <div class="title">External Access URL</div>
      <input spellcheck="false" bind:value={access}/>
    </div>
    <div class="commands">
      <button class="cmd launch">{pier.running ? "Suspend" : "Start"} Ship</button>
      <button class="cmd eject">Eject/Migrate Pier</button>
      <button class="cmd delete">Delete Ship</button>
    </div>
  {/if}
</div>

<style>
  .ship {
    padding: 20px;
    width: 480px;
    max-width: calc(100vw - 40px);
  }
  .card {
    display: flex;
    gap: 20px;
    align-items: end;
    margin-bottom: 24px;
  }
  .sigil {
    width: 87px;
    height: 87px;
    max-height: 87px;
    background: #111;
    border-radius: 15px;
  }
  .status {
    opacity: .8;
    font-weight: 400;
    font-size: .8em;
    padding-bottom: 6px;
    color: red;
  }
  .running {
    color: lime;
  }
  .nick {
    padding-left: 6px;
    font-style: oblique;
    font-size: 14px;
  }
  .patp {
    font-size: 16px;
    padding-bottom: 8px;
  }
  .info {
    display: flex;
    flex-direction: column;
    margin-bottom: 12px;
  }
  .title {
    font-weight: 700;
    margin-bottom: 6px;
    text-align: left;
  }
  input {
    flex: 1;
    padding: 12px;
    font-size: 16px;
    color: inherit;
    font-weight: 700;
    background: #FBFBFB80;
    outline: none;
    border: none;
    border-radius: 6px;
  }
  input:focus {
    background: #EBEBEB80;
  }

  .commands {
    display: flex;
    flex-direction: column;
    gap: 12px;
    padding-top: 12px;
  }
  .cmd {
    background: none;
    color: inherit;
    font-size: 14px;
    font-weight: 600;
    border: none;
    border-radius: 8px;
    padding: 9px;
    width: 180px;
    cursor: pointer;
  }

  .launch {
    background: #008EFF;
  }

  .eject {
    background: #FFFFFF4D;
  }

  .delete {
    background: #f48399;
  }

</style>
