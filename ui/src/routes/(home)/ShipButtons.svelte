<script>
  import { goto } from '$app/navigation'
  import { URBIT_MODE } from '$lib/stores/data'

  export let patp
  export let url

  $: pfx = $URBIT_MODE ? "/apps/groundseg" : ""

</script>
<div class="outer">
  <a href={url} target="_blank">
    <div class="ext-icon"></div>
  </a>
  <button on:click={()=>goto(pfx+"/"+patp)}>
    <div class="settings-icon"></div>
  </button>
</div>
<style>
  .outer {
    display: flex;
    gap: 15px;
  }
  a {
    text-decoration: none;
    width: 48px;
    height: 32px;
    flex-shrink: 0;
    border-radius: 24px 4px 0px 0px;
    background: var(--Gray-400, #5C7060);
    position: relative;
    color: var(--text-card-color);
  }
  .ext-icon {
    width: 16px;
    height: 16px;
    position: absolute;
    left: calc(50% - 4px);
    top: calc(50% - 8px);
    background-image: url('/ArrowUpRight.svg');
    background-repeat: no-repeat;
    background-position: center;
  }
  button {
    text-decoration: none;
    width: 48px;
    height: 32px;
    flex-shrink: 0;
    border-radius: 24px 0px 22px 0px;
    background: var(--Gray-400, #5C7060);
    cursor: pointer;
    position: relative;
  }
  .settings-icon {
    width: 48px;
    height: 32px;
    position: absolute;
    top: 0;
    left: 0;
    background-image: url('/settings-new.svg');
    background-repeat: no-repeat;
    background-size: contain;
  }
</style>
