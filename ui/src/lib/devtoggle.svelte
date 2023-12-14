<script>
  import { devShipClass } from '$lib/stores/devclass'
  import { resetSetup } from '$lib/stores/websocket'
</script>
  <div class="wrapper">
    {"Reset to Setup:".toUpperCase()}
    <button
      on:click={resetSetup}
      >Reset
    </button>
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
  .wrapper {
    display: inline;
    float: right;
  }
  button {
    border: solid 1px black;
    color: black;
  }
  button:hover {
    cursor: pointer;
  }
  .active {
    background: black;
    color: white;
  }

</style>
