<script>
  import Ship from '/src/Home/Ship.svelte'
  import Nav from '/src/Home/Nav.svelte'

  let patps = [
    'dister-dozzod-nallux-dozryl',
    'famwyl-lavlyr-mopfel-winrux',
    'tabtes-sorlec-mopfel-winrux'
  ]

</script>

<svelte:head>
	<title>Home</title>
	<meta name="description" content="Native Planet" />
</svelte:head>

<div class="container">
  <Nav page="home"/>
  <div class="list">
    {#each patps as p}
      <Ship {p} /> 
    {/each}
  </div>
</div>
<style>
  .container {
    margin: auto;
    width: 600px;
    margin-top: 6em;
  }
  .list {
    background-color: rgba(19,19,19,0.6);
    padding: 1em;
    margin-top: 1em;
    border-radius: 12px;
    box-shadow: 0 0 12px #fff;
  }
</style>
