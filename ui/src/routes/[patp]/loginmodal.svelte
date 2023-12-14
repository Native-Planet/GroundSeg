<script>
  import { afterUpdate } from 'svelte'
  import { login } from '$lib/stores/websocket'
  import { URBIT_MODE } from '$lib/stores/data'
  import { gallsegLoginInfo } from '$lib/stores/urbit'
  import { closeModal } from 'svelte-modals'
  import Modal from '$lib/modal.svelte'
  import { goto } from '$app/navigation';

  export let isOpen
  let pwd = ''
  let errMsg = ''

  /*
  afterUpdate(()=>{
    if (tRegister == "done") {
      closeModal()
    }
  })
  */

  let clicked = false

  $: listenLogin($gallsegLoginInfo)

  $: pfx = $URBIT_MODE ? "/apps/groundseg" : ""

  const listenLogin = info => {
    if (clicked) {
      if (info.error.length > 0) {
        errMsg = info.error
        setTimeout(()=>clicked=false,3000)
      } else {
        errMsg = "success"
        setTimeout(()=>{
          closeModal()
          goto(pfx+"/")
        }, 3000)
      }
    }
  }

  const handleLogin = () => {
    clicked = true
    login(pwd)
  }
</script>

{#if isOpen}
  <Modal>
    <div class="wrapper">
      <div class="title">Admin Password</div>
      <input type="password" bind:value={pwd} />
      <button
        disabled={(pwd.length < 1) || clicked}
        on:click={handleLogin}>
        {
          !clicked ? "Login" :
          (errMsg.length < 1) ? "Logging you in..." :
          (errMsg == "success") ? "Success!" :
          errMsg
        }
      </button>
    </div>
  </Modal>
{/if}

<style>
  .wrapper {
    margin: 32px;
    font-family: var(--regular-font);
  }
  .title {
    color: #000;
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    letter-spacing: -1.44px;
  }
  input {
    flex: 1;
    color: var(--NP_Black, #313933);
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    letter-spacing: -1.44px;
    border-radius: 16px;
    background: var(--Gray-100, #DDE3DF);
    padding: 16px 24px 18px 24px;
    border: none;
  }
  input:focus {
    outline: none;
  }
  button {
    margin-top: 20px;
    font-family: var(--regular-font);
    background: var(--btn-secondary);
    color: var(--text-card-color);
    height: 65px;
    font-size: 12px;
    border-radius: 16px;
    cursor: pointer;
    justify-content: center;
    align-items: center;
    padding: 0 48px;
    cursor: pointer;

    color: #FFF;
    text-align: center;
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    line-height: 32px; /* 133.333% */
    letter-spacing: -1.44px;
  }
  button:disabled {
    opacity: .6;
    pointer-events: none;
  }
</style>
