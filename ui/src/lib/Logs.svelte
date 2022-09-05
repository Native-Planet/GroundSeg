<script>
  import { onMount } from 'svelte'
  import { api } from '$lib/api'
  import Select from 'svelte-select'
  import Fa from 'svelte-fa'

  let container = null, logs = [], data = []

  onMount(() => {
    fetch(api + "/settings/logs").then(r => r.json()).then(d => {
      for (let i = 0; i < d.length; i++) {
        let value = d[i]
        let label = d[i].charAt(0).toUpperCase() + d[i].slice(1)
        logs[i] = {value: value, label: label}
      }
      data = d
  })})

  const inc = c => !(data.includes(c))

  const exportLog = c => {
    const u = url + "/settings/logs"
    const f = new FormData()
    f.append('logs', container)
    fetch(u, {method: 'POST', body: f})
      .then(r => r.json()).then(d => {
          var element = document.createElement('a')
          element.setAttribute('href', 'data:text/plain;charset=utf-8,' + encodeURIComponent(d))
          element.setAttribute('download', container)
          element.style.display = 'none'
          document.body.appendChild(element)
          element.click()
          document.body.removeChild(element)
    })}

</script>
  <div class="network">
    <div class="network-title">Export Logs</div>
    <div class="select">
      <Select
          items={logs}
          listPlacement="top"
          on:clear={()=> container = null}
          on:select={e => container = e.detail.value} />
    </div>
    <div class="buttons">
      <a class="view" class:disabled={inc(container)} href="/logs/{container}#latest">View</a>
      <button
        on:click={exportLog(container)}
        class="export"
        class:disabled={inc(container)}>Export</button>
    </div>
  </div>

<style>
  .network {
    display: flex;
    flex-direction: column;
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
    appearance: none;
  }
  .buttons {
    display: flex;
  }
  .view {
    background: #ffffff4d;
    padding: 6px;
    font-size: 12px;
    border-radius: 6px;
    width: 60px;
    text-align: center;
  }
  .export {
    font-weight: 700;
    margin-left: auto;
    background: #008eff;
    padding: 6px;
    font-size: 12px;
    border-radius: 6px;
    width: 80px;
    text-align: center;
    cursor: pointer;
    border: none;
    color: white;
  }

  .disabled {
    opacity: .6;
    pointer-events: none;
  }
</style>
