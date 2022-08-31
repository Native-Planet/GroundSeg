<script>
  import { onMount } from 'svelte'
  import { url } from '/src/Scripts/server'
  import { page } from '$app/stores';
  import Logo from '/src/Components/Buttons/Logo.svelte'
  import DeleteWarning from '/src/Components/DeleteWarning.svelte'
  import Sigil from '/src/Components/Sigil.svelte'
  import PierCredentials from '/src/Components/PierCredentials.svelte'

  const cur = $page.url;
  const path = cur.pathname.replace("/", "")

  let loading = false,
    ejecting = false,
    deleteCheck = false,
    data = {'nw_label': '', 'pier':{}}

  onMount(async () => {
    getPierData()
  })
  
  const getPierData = () => {
    const u = url + "/urbit/pier?pier=" + path
    fetch(u).then(r => r.json()).then(d => data = d)
  }

  const ejectPier = () => {
    ejecting = true
    let u = url + "/urbit/eject"
    const f = new FormData()
    f.append(data.pier.name, 'eject')

    fetch(u, {method: 'POST',body: f})
    .then(res => { return res.blob(); })
    .then(d => {
      ejecting = false
      var a = document.createElement("a")
      a.href = window.URL.createObjectURL(d)
      a.download = data.pier.name
      a.click()
    })}

  const toggleNetwork = () => { console.log("POST placeholder") }

  const togglePier = () => {
    loading = true
    let u = url + "/urbit/"
    const f = new FormData()
    if (data.pier.running) {
      f.append(data.pier.name, 'stop')
      u = u + 'stop'
    }

    if (!data.pier.running) {
      f.append(data.pier.name, 'start')
      u = u + 'start'
    }

    fetch(u, {method: 'POST',body: f})
      .then(r => r.json())
      .then(d => {
        if (d == 200) {
          getPierData()
          loading = false
      }})
  }

  const deletePier = () => {
    let u = url + "/urbit/delete"
    const f = new FormData()
    f.append(data.pier.name, 'delete')

    fetch(u, {method: 'POST',body: f})
      .then(r => r.json())
      .then(d => { if (d == 200) {
        window.location.href = "/"
   }})}


</script>
<Logo t="Pier Settings" />
<div class="ship">
  {#if data}
  {#if deleteCheck}
    <DeleteWarning on:back={()=>deleteCheck = false} on:delete={deletePier} name={data.pier.name} />
  {:else}
    <div class="card">
      <Sigil patp={data.pier.name} size="87px" rad="15px" />
      <div class="info">
        <div class="status {data.pier.running ? "running" : ""}">
          {data.pier.running ? "Running" : "Stopped"} 
        </div>
        <div class="patp">{data.pier.name}</div>
      </div>
    </div>
    {#if data.pier.running}
      <PierCredentials
        code={data.pier.code}
        ext={data.pier.url}
        nw_label={data.nw_label} />
    {/if}
    <div class="commands">
      <button on:click={togglePier} class="cmd launch">
        {data.pier.running ? "Suspend" : "Start"}{loading ? "ing" : " Ship"}
      </button>
      <button 
        on:click={ejectPier}
        class="cmd eject">
        Eject{ejecting ? "ing" : " Pier"}
      </button>
      <button on:click={()=> deleteCheck = true} class="cmd delete">Delete Pier</button>
    </div>
  {/if}
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
  .commands {
    display: flex;
    flex-direction: column;
    gap: 12px;
    padding-top: 18px;
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
