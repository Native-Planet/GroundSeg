<script>
  import { devShipClass } from '$lib/stores/devclass'
</script>
<div>
  {"Dev Mode Ship Class:".toUpperCase()}
  <button
    on:click={()=>devShipClass.set("star")}
    class:active={$devShipClass == "star"}
    >Star
  </button>
  <button
    on:click={()=>devShipClass.set("planet")}
    class:active={$devShipClass == "planet"}
    >Planet
  </button>
  <button
    on:click={()=>devShipClass.set("default")}
    class:active={$devShipClass == "default"}
    >Default
  </button>
</div>

<style>
  div {
    display: inline;
    float: right;
  }
  button {
    border: solid 1px black;
    color: black;
  }
  .active {
    background: black;
    color: white;
  }

</style>
