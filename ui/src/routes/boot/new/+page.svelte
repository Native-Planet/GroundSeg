<script>
  import Logo from '/src/Components/Buttons/Logo.svelte'
  import NewPier from '/src/Components/Buttons/NewPier.svelte'

  let name, key

</script>

<Logo />
<div class="key">
  <div class="info">
    <div class="title">Pier Name</div>
    <input spellcheck="false" bind:value={name}/>
  </div>
  <div class="info">
    <div class="title">Pier Key</div>
    <input spellcheck="false" bind:value={key}/>
  </div>
  <NewPier {name} {key}/>
</div>
<style>
  .key {
    display: flex;
    flex-direction: column;
    gap: 18px;
    color: inherit;
    padding: 20px;
    width: 460px;
    max-width: calc(80vw - 40px);
  }
  .info {
    display: flex;
    flex-direction: column;
  }
  .title {
    font-weight: 700;
    margin-bottom: 6px;
    text-align: left;
  }
  input {
    flex: 1;
    padding: 12px;
    font-size: 16px;
    color: inherit;
    font-weight: 700;
    background: #FBFBFB80;
    outline: none;
    border: none;
    border-radius: 6px;
  }
  input:focus {
    background: #EBEBEB80;
  }
</style>
