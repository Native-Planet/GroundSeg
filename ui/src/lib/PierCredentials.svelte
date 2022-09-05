<script>
  import { api } from '$lib/api'
  import Fa from 'svelte-fa'
  import { faArrowUpRightFromSquare } from '@fortawesome/free-solid-svg-icons/index.es'

  export let name, nw_label, code, ext, minIO

  let viewLogin = false, clickedLogin = false,
    viewExt = false, clickedExt = false,
    viewMinIO = false, clickedMinIO = false,
    isSwitching = false


  // Copy String to Clipboard

  const onCopyLogin = () => {
    navigator.clipboard.writeText(code)  
    clickedLogin = true
    setTimeout(()=> clickedLogin = false, 1000)
  }
  const onCopyExt = () => {
    navigator.clipboard.writeText(ext)  
    clickedExt = true
    setTimeout(()=> clickedExt = false, 1000)
  }

  const onCopyMinIO = () => {
    navigator.clipboard.writeText(minIO)  
    clickedMinIO = true
    setTimeout(()=> clickedMinIO = false, 1000)
  }

  // Network switching

  const toggleNetwork = () => { 
    isSwitching = true
    let u = api + "/urbit/network"
    const f = new FormData()
    f.append(name,'network')

    fetch(u, {method: 'POST',body: f})
      .then(r => r.json())
      .then(d => { if (d == 200) {
        isSwitching = false
   }})}


</script>
    <div class="info">
      <div class="title">Login Key</div>
      <div class="login-key-wrapper">
        <div on:click={onCopyLogin} class="login-key">
          {
            clickedLogin ? "copied!" 
            : viewLogin ? code
            : "click to copy"
          }
        </div>
        <button on:click={()=> viewLogin = !viewLogin}>
          <img class="eye" src={viewLogin ? "/eye-closed.svg" : "/eye-open.svg"} alt="eye" />
        </button>
      </div>
    </div>

    <div class="info">
      <div class="title">External Access URL</div>
      <div class="login-key-wrapper">
        <div on:click={onCopyExt} class="login-key">
          {
            clickedExt ? "copied!" 
            : viewExt ? ext
            : "click to copy"
          }
        </div>
        <a class="newtab" href={ext} target="_blank">
          <Fa icon={faArrowUpRightFromSquare} size="1.2x" />
        </a>
        <button on:click={()=> viewExt = !viewExt}>
          <img class="eye" src={viewExt ? "/eye-closed.svg" : "/eye-open.svg"} alt="eye" />
        </button>
      </div>
    </div>

    <div class="info">
      <div class="title">MinIO Console</div>
      <div class="login-key-wrapper">
        <div on:click={onCopyMinIO} class="login-key">
          {
            clickedMinIO ? "copied!" 
            : viewMinIO ? minIO
            : "click to copy"
          }
        </div>
        <a class="newtab" href={minIO} target="_blank">
          <Fa icon={faArrowUpRightFromSquare} size="1.2x" />
        </a>
        <button on:click={()=> viewMinIO = !viewMinIO}>
          <img class="eye" src={viewMinIO ? "/eye-closed.svg" : "/eye-open.svg"} alt="eye" />
        </button>
      </div>
    </div>

    <div class="info"class:switching={isSwitching} on:click={toggleNetwork}>
      <div class="title">Access</div>
      <div class="access-options">
        <button class="option" class:access-active={nw_label === 'Local'} >Local</button>
        <button class="option" class:access-active={nw_label === 'Remote'} >Remote</button>
      </div>
    </div>

<style>
  button {
    appearance: none;
    background: none;
    border: none;
    padding: 0;
    margin: 0;
    height: 32px;
  }
  .info {
    margin-bottom: 12px;
  }
  .title {
    font-weight: 700;
    margin-bottom: 12px;
    text-align: left;
  }
  .login-key-wrapper {
    display: flex;
  }
  .login-key {
    font-style: italic;
    font-size: 12px;
    padding: 8px;
    background: #ffffff4d;
    border-radius: 6px;
    flex: 1;
  }
  .eye {
    height: 32px;
    opacity: .8;
    margin-left: 12px;
    cursor: pointer;
  }
  .newtab {
    margin: auto;
    margin-left: 16px;
    opacity: .8;
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
    cursor: pointer;
  }
  .switching {
    opacity: .6;
    pointer-events: none;
  }
  .access-active {
    background: #008eff;
  }

</style>
