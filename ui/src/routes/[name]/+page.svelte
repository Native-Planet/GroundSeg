<script>
  import { onMount, onDestroy } from 'svelte'
  import { api } from '$lib/api'
  import { page } from '$app/stores'
  import { profile } from '$lib/components'
  import Fa from 'svelte-fa'
  import { faChevronDown, faChevronUp } from '@fortawesome/free-solid-svg-icons/index.es'
  import Logs from '$lib/Logs.svelte'
  import PrimaryButton from '$lib/PrimaryButton.svelte'
  import UpdateInstructions from '$lib/UpdateInstructions.svelte'
  import Clipboard from 'clipboard'

  const cur = $page.url;
  const path = cur.pathname.replace("/", "")

  let loading = false,
    ejecting = false,
    deleteCheck = false,
    data = {'nw_label': '', 'pier':{}},
    advanced = false,
    shown = true,
    showLogs = false,
    code = '',
    clickedPatp = false,
    fresh = true
  
  let copyPatp

  onMount(() => {
    fresh = true
    getPierCode()
    getPierData()
    copyPatp = new Clipboard('#patp')
    copyPatp.on("success", ()=> {
    clickedPatp = true; setTimeout(()=> clickedPatp = false, 1000)})
    setTimeout(()=> fresh = false, 1000)
  })
  onDestroy(() => shown = false)
  
  const getPierData = () => {
    if (shown) {
      const u = api + "/urbit/pier?pier=" + path
      fetch(u).then(r => r.json()).then(d => handleData(d))
      setTimeout(getPierData, 1000)
    }
  }

  const getPierCode = () => {
    if (shown) {
      const u = api + "/urbit/code?pier=" + path
      fetch(u).then(r => r.json()).then(d => {
        code = d
        if (d.length == 27) {setTimeout(getPierCode, 1800000)}
        else {setTimeout(getPierCode, 1000)}})}}

  const handleData = d => {
    if (d == 400) { window.location.href = "/" }
    if (d.pier.name == path) { data = d }}

  const toggleLogs = () => showLogs = !showLogs

  const ejectPier = () => {
    ejecting = true
    let u = api + "/urbit/eject"
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
    let u = api + "/urbit/"
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
    let u = api + "/urbit/delete"
    const f = new FormData()
    f.append(data.pier.name, 'delete')

    fetch(u, {method: 'POST',body: f})
      .then(r => r.json())
      .then(d => { if (d == 200) {
        window.location.href = "/"
   }})}

  const updateMinIO = () => {
    let u = api + "/urbit/minio_endpoint"
    const f = new FormData()
    f.append('pier', data.pier.name)

    fetch(u, {method: 'POST',body: f})
      .then(r => r.json())
      .then(d => { if (d == 200) {
        console.log('updated minio endpoints in ship')}
        else {
        console.log('failed to update minio endpoints')}})}

</script>

<div class="mega-wrapper">

  {#if showLogs}

    <!-- Logs -->
    <Logs log={data.pier.name} maxHeightOffset={100}/>
    <div class="bottom-panel">
      <svelte:component this={profile.sigil}  patp={data.pier.name} size="60px" rad="8px" />
      <div class="info">
        {#if loading}
          <div class="status loading">Loading...</div>
        {:else if data.pier.running}
          {#if fresh}
            <div classs="status loading">Loading...</div>
          {:else if code.length != 27}
            <div class="status booting">Booting</div>
          {:else}
            <div class="status running">Running</div>
          {/if}
        {:else}
          <div class="status">Stopped</div>
        {/if}
        <div class="patp-logs">{data.pier.name}</div>
      </div>

      <PrimaryButton standard="Back to profile" status="standard" left={false} on:click={toggleLogs} />
    </div>

  {:else}
    
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
                {#if fresh}
                  <div class="status loading">Loading...</div>
                {:else if code.length != 27}
                  <div class="status booting">Booting</div>
                {:else}
                  <div class="status running">Running</div>
                {/if}
              {:else}
                <div class="status">Stopped</div>
              {/if}

              <div
                on:click={copyPatp}
                data-clipboard-text={data.pier.name}
                id="patp"
                class="patp">
                {clickedPatp ? "copied!" : data.pier.name}
              </div>
            </div>
          </div>

          <!-- Pier Credentials -->
          {#if (code.length == 27) && data.pier.running}
            <svelte:component this={profile.credentials}
              minIO={data.pier.s3_url}
              name={data.pier.name}
              nw_label={data.nw_label}
              code={code}
              ext={data.pier.url}
              wg_running={data.wg_running}
              minIO_reg={data.pier.minio_registered}
              wg_reg={data.wg_reg} />
          {/if}
      
          <div class="commands">
            <div class="advanced" on:click={()=> advanced = !advanced}>
              Advanced Options
              <Fa icon={advanced ? faChevronUp : faChevronDown} size="0.8x" />
          </div>

          {#if advanced}
            <div class="cmd-wrapper">
              <button
                on:click={toggleLogs} 
                class="cmd logs">
                View Logs
              </button>

            {#if data.pier.minio_registered && (data.nw_label == 'Remote')}
              <button
                on:click={updateMinIO} 
                class="cmd minio">
                Update minIO
              </button>
            {/if}

              <button 
                on:click={ejectPier}
                class="cmd eject">
                Eject{ejecting ? "ing" : " Pier"}
              </button>

              <button on:click={()=> deleteCheck = true} class="cmd delete">Delete Pier</button>
            </div>
          {/if}
        </div>
      {/if}

    {:else}
      <div class="block"></div>
    {/if}
  </div>
  {/if} <!-- End #if show logs -->
</div> <!-- End mega wrapper -->

<style>
  @keyframes breathe {
    0% {opacity: .6}
    50% {opacity: 0}
    100% {opacity: .6}
  }
  .mega-wrapper {
    max-height: 80vh;
    overflow: auto;
    -ms-overflow-style: none;
    scrollbar-width: none;
  }
  .mega-wrapper::-webkit-scrollbar {
    display: none;
  }
  .bottom-panel {
    margin: 20px;
    display: flex;
    gap: 18px;
    align-items: center;
  }
  .ship {
    padding: 20px;
    width: 480px;
    max-width: calc(100vw - 40px);
  }
  .card {
    display: flex;
    gap: 20px;
    align-items: end;
    margin-bottom: 14px;
  }
  .switch-wrapper {
    cursor: pointer;
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
    cursor: pointer;
  }
  .patp-logs {
    font-size: 12px;
  }
  .info {
    display: flex;
    flex-direction: column;
    margin-bottom: 12px;
  }
  .commands {
    padding-top: 6px;
  }
  .cmd-wrapper {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }
  .cmd {
    appearance: none;
    background: none;
    color: inherit;
    font-size: 12px;
    font-weight: 700;
    border: none;
    border-radius: 8px;
    padding: 8px;
    width: 120px;
    cursor: pointer;
  }

  .advanced {
    font-size: 14px;
    padding-top: 6px;
    padding-bottom: 24px;
    cursor: pointer;
    width: 150px;
  }
  .advanced:hover {
    opacity: .6;
  }
  .logs {
    background: var(--action-color);
  }
  .minio {
    background: orange;
  }
  .eject {
    background: #FFFFFF4D;
  }

  .delete {
    background: #f48399;
  }
  .block {
    background: #ffffff4d;
    height: 158px;
    width: 100%;
    filter: blur(20px);
    animation: breathe 2s infinite;
  }
</style>
