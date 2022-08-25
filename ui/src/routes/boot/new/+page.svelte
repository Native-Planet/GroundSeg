<script>
  import Nav from '/src/Home/Nav.svelte'
</script>

<div class="container">
  <Nav page="Boot a new Urbit ID" />
</div>
<div class="container inputs">
  <div class="title">Pier Name</div>
  <input />
  <div class="title">Pier Key</div>
  <input />
  <button class="submit">Submit</button>
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
  input {
    margin: .6em 20px .6em 20px;
  }
  .submit {
    padding: 1em;
    background: #443cf4;
    margin: 2em 20px .6em 20px;
  }
  .submit:hover {
    color: inherit;
    opacity: .8;
  }
  .submit:active {
    opacity: .6;
  }
</style>
