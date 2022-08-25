<script>
  import Nav from '/src/Home/Nav.svelte'
</script>

<div class="container">
  <Nav page="Boot an existing Urbit ID" />
</div>
<div class="container upload">
  <div class="title">Upload a pier folder</div>
  <div class="subtitle">Should be zip'd or tar'd. When finished please refresh page.</div>
  <div class="drop">
    <div class="text">Drop files here to upload</div>
  </div>
</div>
<style>
  .container {
    margin: auto;
    width: 600px;
    margin-top: 6em;
  }
  .upload {
    background-color: rgba(19,19,19,0.6);
    margin-top: 1em;
    border-radius: 12px;
    box-shadow: 0 0 12px #fff;
    color: inherit;
    padding: 1em;
    width: calc(600px - 2em);
  }
  .title {
    font-size: 1.2em;
    font-weight: 600;
    padding: .3em 0 0 20px;
    text-align: left;
  }
  .subtitle {
    padding: .6em 0 0 20px;
    opacity: .8;
    font-weight: 600;
  }
  .drop {
    position: relative;
    margin: 2em 2em 1em 2em;
    height: 8em;
    text-align: center;
    border: 1px solid #444;
  }
  .text {
    position: absolute;
    top: calc(50% - .5em);
    width: 100%;
    margin: auto;
  }
</style>
