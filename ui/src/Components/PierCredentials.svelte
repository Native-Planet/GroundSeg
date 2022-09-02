<script>
  import { url } from '/src/Scripts/server'
  import Fa from 'svelte-fa'
  import { faArrowUpRightFromSquare } from '@fortawesome/free-solid-svg-icons/index.es'

  export let code, ext

  let viewLogin = false, clickedLogin = false,
    viewExt = false, clickedExt = false,
    viewMinIO = false, clickedMinIO = false

  // placeholder
  let minIO = "placeholder"

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
      <div class="title">MinIO Bucket</div>
      <div class="login-key-wrapper">
        <div on:click={onCopyMinIO} class="login-key">
          {
            clickedMinIO ? "copied!" 
            : viewMinIO ? minIO
            : "click to copy"
          }
        </div>
        <button on:click={()=> viewMinIO = !viewMinIO}>
          <img class="eye" src={viewMinIO ? "/eye-closed.svg" : "/eye-open.svg"} alt="eye" />
        </button>
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
    background: #FBFBFB80;
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
</style>
