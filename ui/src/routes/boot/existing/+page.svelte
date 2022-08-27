<script>
  import Logo from '/src/Components/Buttons/Logo.svelte'
</script>

<Logo />
<div class="pier">
  <div class="title">Upload a pier folder</div>
  <div class="subtitle">Should be zip'd or tar'd. When finished please refresh page.</div>
  <div class="drop">
    <div class="text">Drop files here to upload</div>
  </div>
  <a href="/">Cancel</a>
</div>
<style>

  .pier {
    display: flex;
    flex-direction: column;
    color: inherit;
    padding: 20px;
    width: 460px;
    max-width: calc(80vw - 40px);
  }

  .title {
    font-size: 1.3em;
    font-weight: 700;
    padding-bottom: 6px;
    text-align: left;
  }
  .subtitle {
    opacity: .7;
    font-weight: 700;
    font-size: .9em;
    padding-bottom: 24px;
  }
  .drop {
    position: relative;
    height: 120px;
    text-align: center;
    border: 1px solid #ffffff80;
  }
  .text {
    position: absolute;
    top: calc(50% - .5em);
    width: 100%;
    margin: auto;
  }
  a {
    margin-top: 24px;
    padding: 12px 20px 12px 20px;
    color: inherit;
    text-align: center;
    font-weight: 500;
    border-radius: 8px;
    background: #FFFFFF4D; 
    font-size: 16px;
  }

</style>
