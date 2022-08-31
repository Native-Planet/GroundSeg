<script>
  import { onMount } from 'svelte'
  import { url } from '/src/Scripts/server'
  import { page } from '$app/stores';
  import Logo from '/src/Components/Buttons/Logo.svelte'

  const cur = $page.url;
  const path = cur.pathname.replace("/", "")
  let access, key, minIO
  let data = {'nw_label': '', 'pier':{}}

  onMount(async () => {
    const u = url + "/urbit/pier?pier=" + path
    fetch(u).then(r => r.json()).then(d => data = d)
  })

  const toggleNetwork = () => { console.log("POST placeholder") }
  const togglePier = () => {
    const u = url + "/urbit/"
    const f = "dist"
    const r = fetch(u, {method: 'POST',body: f})

  }

</script>
<Logo t="Pier Settings" />
<div class="ship">
  {#if data}
    <div class="card">
      <div class="sigil"></div>
      <div class="info">
        <div class="status {data.pier.running ? "running" : ""}">
          {data.pier.running ? "Running" : "Stopped"} 
        </div>
        <div class="patp">
          {data.pier.name}
        </div>
      </div>
    </div>
    {#if data.pier.running}
    <div class="info">
      <div class="title">Login Key</div>
      <input spellcheck="false" type="password" bind:value={key}/>
    </div>
    <div class="info">
      <div class="title">External Access URL</div>
      <input spellcheck="false" bind:value={access}/>
    </div>
    <div class="info">
      <div class="title">MinIO Bucket</div>
      <input spellcheck="false" bind:value={minIO}/>
    </div>
    <div class="info">
      <div class="title">Access</div>
      <div class="access-options">
        <button class="option" class:access-active={data.nw_label === 'Local'} on:click={toggleNetwork}>Local</button>
        <button class="option" class:access-active={data.nw_label === 'Remote'} on:click={toggleNetwork}>Remote</button>
      </div>
    </div>
    {/if}
    <div class="commands">
      <button on:click={togglePier} class="cmd launch">{data.pier.running ? "Suspend" : "Start"} Ship</button>
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
    padding: 8px;
    font-size: 12px;
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
  .access-options {
    display: flex;
    width: 240px;
    border-radius: 8px;
    background: #ffffff4d;
    gap: 2px;
  }
  .option {
    color: inherit;
    font-size: 14px;
    flex: 1;
    padding: 8px 0 8px 0;
    background: none;
    border-radius: 8px;
    border: none;
    font-weight: 700;
  }
  .access-active {
    background: #008eff;
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
