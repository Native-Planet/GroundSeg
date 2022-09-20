<script>
  import { newID } from '$lib/components'
  let name = '',
    key = '',
    minio = '',
    viewKey = false

  const toggleViewKey = () => {
    viewKey = !viewKey
    document.querySelector('#key').type = viewKey ? 'text' : 'password'
  }

</script>

<svelte:component this={newID.logo} />
<div class="key">
  <div class="info">
    <div class="title">Pier Name</div>
    <input spellcheck="false" placeholder="sampel-palnet" bind:value={name}/>
  </div>

  <div class="info">
    <div class="title">Pier Key</div>
    <div class="pass-wrapper">
      <input spellcheck="false" id="key" type="password" bind:value={key}/>
      <img on:click={toggleViewKey} src="/eye-{viewKey ? "closed" : "open"}.svg" alt="eye" />
    </div>
  </div>

  <svelte:component this={newID.newPier} {name} {key} />

</div>
<style>
  .key {
    display: flex;
    flex-direction: column;
    gap: 24px;
    color: inherit;
    padding: 20px;
    width: 460px;
    max-width: calc(100vw - 40px);
  }
  .info {
    display: flex;
    flex-direction: column;
  }
  .title {
    font-family: inherit;
    font-size: 13px;
    font-weight: 700;
    margin-bottom: 8px;
    text-align: left;
  }
  .pass-wrapper {
    display: flex;
  }
  input {
    flex: 1;
    padding: 8px;
    font-size: 12px;
    color: inherit;
    font-weight: 700;
    background: #ffffff4d;
    outline: none;
    border: none;
    border-radius: 6px;
  }
  .pass-wrapper > img {
    padding-left: 12px;
  }

</style>
