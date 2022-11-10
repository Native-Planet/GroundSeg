<script>
  import { createEventDispatcher } from 'svelte'
  import Fa from 'svelte-fa'
  import { faChevronDown, faChevronUp } from '@fortawesome/free-solid-svg-icons'
  export let advanced
  const dispatch = createEventDispatcher()
</script>
<div class="advanced" on:click={()=>dispatch('click')} >
  Advanced Options
  <Fa icon={advanced ? faChevronUp : faChevronDown} size="0.8x" />
</div>

<style>
  .advanced {
    padding-top: 6px;
    color: inherit;
    font-size: 12px;
    cursor: pointer;
    width: 123px;
  }

</style>
