<script>
  import Nav from '/src/Home/Nav.svelte'
</script>

<div class="container">
  <Nav page="System Settings" />
</div>
<div class="container inputs">
  <div class="title">Todo</div>
</div>
<style>
  .container {
    margin: auto;
    width: 600px;
    margin-top: 6em;
  }
  .inputs {
    display: flex;
    flex-direction: column;
    background-color: rgba(19,19,19,0.6);
    margin-top: 1em;
    border-radius: 12px;
    box-shadow: 0 0 12px #fff;
    color: inherit;
    padding: 1em;
    width: calc(600px - 2em);
  }
  .title {
    font-size: 1em;
    font-weight: 600;
    padding: .3em 0 0 20px;
    text-align: left;
  }
</style>
