<script>
  import { onMount, onDestroy } from 'svelte'
  import { url } from '/src/Scripts/server'
  import { page } from '$app/stores'
  import { profile } from '$lib/components'
  import Fa from 'svelte-fa'
  import { faChevronDown, faChevronUp } from '@fortawesome/free-solid-svg-icons/index.es'

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

<svelte:component this={profile.logo} t="Pier Settings" />

<div class="ship">
  {#if data.pier.name != undefined}

    {#if deleteCheck}

      <svelte:component
        this={profile.warning}
        on:back={()=>deleteCheck = false}
        on:delete={deletePier}
        name={data.pier.name} />

  {:else}

    <div class="card">

      <svelte:component this={profile.sigil}  patp={data.pier.name} size="87px" rad="15px" />

      <div class="info">
        <div on:click={togglePier} class="switch-wrapper">
          <div class="switch {data.pier.running ? "on" : "off"}"></div>
        </div>

        {#if loading}

          <div class="status loading">Loading...</div>

        {:else if data.pier.running}

          {#if (data.pier.code == undefined || data.pier.code == '')}

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
    {#if data.pier.running && (!(data.pier.code == undefined) && !(data.pier.code == ""))}
      <svelte:component this={profile.credentials}
        name={data.pier.name}
        nw_label={data.nw_label}
        code={data.pier.code}
        ext={data.pier.url} />
    {/if}
    <div class="commands">
      <span class="advanced" on:click={()=> advanced = !advanced}>
        Advanced Options
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
  .switch-wrapper {
    border-radius: 8px;
    width: 32px;
    height: 12px;
    background: #ffffff4d;
    padding: 2px;
    margin-bottom: 6px;
  }
  .switch {
    height: 100%;
    width: 19px;
    border-radius: 6px;
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
  .status {
    opacity: .8;
    font-weight: 400;
    font-size: .8em;
    padding-bottom: 6px;
    color: red;
  }
  .loading {
    color: inherit;
    font-style: italic;
  }
  .booting {
    color: orange;
  }
  .running {
    color: lime;
  }
  .patp {
    font-size: 16px;
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

  .eject {
    background: #FFFFFF4D;
  }

  .delete {
    background: #f48399;
  }

</style>
