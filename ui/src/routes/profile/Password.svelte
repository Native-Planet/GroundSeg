<script>
  // Modals
  import { openModal } from 'svelte-modals'
  import PasswordModal from './PasswordModal.svelte'
  // Styling
  export let wide
  // Websocket
  import { structure, logout } from '$lib/stores/websocket'

  let clicked = false
  const handleLogout = () => {
    clicked = true
    logout()
  }
</script>

<div class="container {wide ? "wide" : "slim"}">
  <div class="prof-title">COMPUTER</div>
  <div class="label">Password</div>
  <div class="wrapper">
    <div class="pwd">****************</div>
    <button class="edit" on:click={()=>openModal(PasswordModal)}>Edit</button>
  </div>
  <div class="label">Session</div>
  <button class="logout" on:click={handleLogout}>{clicked ? "Logging out..." : "Logout"}</button>
</div>

<style>
  .prof-title {
    margin-bottom: 56px;
  }
  .container {
    margin: auto;
  }
  .wrapper {
    display: flex;
    align-items: center;
    gap: 8px;
    height: 65px;
    margin-bottom: 48px;
  }
  .pwd {
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
    padding: 22px 24px 14px 24px;
  }
  .pwd:focus {
    outline: none;
  }
  .label {
    color: var(--Gray-400, #5C7060);
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    line-height: 48px; /* 200% */
    letter-spacing: -1.44px;
  }
  .edit {
    border-radius: 16px;
    background: var(--Gray-400, #5C7060);
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
    height: 65px;
    padding: 0 48px;
    cursor: pointer;
  }
  .logout {
    border-radius: 16px;
    background: black;
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
    height: 65px;
    padding: 0 48px;
    cursor: pointer;
  }
  .wide {
    width: 992px;
    max-width: 98vw;
    padding: 56px;
  }
  .slim {
    width: 100vw;
  }
</style>
