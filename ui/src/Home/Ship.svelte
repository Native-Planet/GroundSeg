<script>
  import Fa from 'svelte-fa/src/fa.svelte'
  import { faGear } from '@fortawesome/free-solid-svg-icons'
  let gear = false;

  export let p;

</script>
<div class="wrapper">
  <div class="ship">
    <div class="sigil off">
    </div>
    <div class="patp">~{p}</div>
    <button class="gear" class:open={gear} on:click={()=> gear = !gear}>
      <Fa icon={faGear} size="1.2x" />
    </button>
  </div>
  {#if gear}
    <div class="cmd">
      <button class="sub start">Start</button>
      <button class="sub eject">Eject</button>
      <button class="sub delete">Delete</button>
    </div>
  {/if}
</div>

<style>
  .wrapper {
    padding: 1em;
  }
  .ship {
    display: flex;
    align-items: center;
  }
  .sigil {
    width: 60px;
    height: 60px;
    background: grey;
    border-radius: 12px;
  }
  .patp {
    flex: 1;
    padding-left: .6em;
  }
  .open {
    color: grey;
  }
  .cmd {
    display: flex;
    justify-content: center;
    gap: .3em;
    padding: .6em;
    border-bottom: solid 1px #555;
  }
  .sub {
    width: 4em;
    padding: .3em;
    border-radius: 4px;
  }
  .sub:hover {
    opacity: .8;
    color: white;
  }
  .sub:active {
    opacity: .6;
  }
  .start {
    background: #5cdb5c;
  }
  .eject {
    background: #ffa500;
  }
  .delete {
    background: #ff0021;
  }
</style>
