<script>
  import { onMount, onDestroy } from 'svelte'
  import { url } from '/src/Scripts/server'
  import { page } from '$app/stores';
  import Fa from 'svelte-fa'
  import { faChevronDown, faChevronUp } from '@fortawesome/free-solid-svg-icons/index.es'

  import Logo from '/src/Components/Buttons/Logo.svelte'
  import DeleteWarning from '/src/Components/DeleteWarning.svelte'
  import Sigil from '/src/Components/Sigil.svelte'
  import PierCredentials from '/src/Components/PierCredentials.svelte'
  import AccessToggle from '/src/Components/AccessToggle.svelte'

  const cur = $page.url;
  const path = cur.pathname.replace("/", "")

  let loading = false,
    ejecting = false,
    deleteCheck = false,
    data = {'nw_label': '', 'pier':{}},
    advanced = false,
    shown = true

  onMount(() => getPierData())
  onDestroy(() => shown = false)
  
  const getPierData = () => {
    if (shown) {
      const u = url + "/urbit/pier?pier=" + path
      fetch(u).then(r => r.json()).then(d => data = d)
      setTimeout(getPierData, 1000)
    }
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
  {#if data.pier.name != undefined}
  {#if deleteCheck}
    <DeleteWarning on:back={()=>deleteCheck = false} on:delete={deletePier} name={data.pier.name} />
  {:else}
    <div class="card">
      <Sigil patp={data.pier.name} size="87px" rad="15px" />
      <div class="info">
        {#if data.pier.running}
          {#if data.pier.code === undefined }
            <div class="status booting">Booting</div>
          {:else}
            <div class="status running">Running</div>
          {/if}
        {:else}
          <div class="status">Stopped</div>
        {/if}
        <div class="patp">{data.pier.name}</div>
      </div>
    </div>
    {#if data.pier.running && data.pier.code !== undefined}
      <PierCredentials
        code={data.pier.code}
        ext={data.pier.url} />
    {/if}
    <AccessToggle 
      name={data.pier.running ? data.pier.name : "Local"}
      nw_label={data.nw_label} />
    <div class="commands">
      <button on:click={togglePier} class="cmd launch">
        {data.pier.running ? "Suspend" : "Start"}{loading ? "ing" : " Ship"}
      </button>
      <span class="advanced" on:click={()=> advanced = !advanced}>
        Advance Options
        <Fa icon={advanced ? faChevronUp : faChevronDown} size="0.8x" />
      </span>
      {#if advanced}
        <button 
          on:click={ejectPier}
          class="cmd eject">
          Eject{ejecting ? "ing" : " Pier"}
        </button>
        <button on:click={()=> deleteCheck = true} class="cmd delete">Delete Pier</button>
      {/if}
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
  .booting {
    color: orange;
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

  .advanced {
    font-size: 14px;
    padding-top: 6px;
    padding-bottom: 6px;
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
