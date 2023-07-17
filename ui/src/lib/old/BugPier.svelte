<script>
  import { createEventDispatcher } from 'svelte'
  import Fa from 'svelte-fa'
  import { faCheck } from '@fortawesome/free-solid-svg-icons'

  export let name = ''
  export let checkAll = false
  export let submitting = false
  export const forceSet = b => {
    check = b
    dispatch('update', {name:name,check:check})
  }

  let check = false

  const dispatch = createEventDispatcher()

  const handleCheck = () => {
    check = !check
    dispatch('update',{name:name,check:check})
  }

</script>

<!-- Checkbox -->
<div class="checker" class:freeze={submitting}>
  <div class="box" class:highlight={check} on:click={handleCheck}>
    {#if check}
      <Fa icon={faCheck} size="1x"/>
    {/if}
  </div>
  <span on:click={handleCheck}>
    {
      !checkAll ? name
      : check ? 'Unselect all'
      : 'Select all'
    }
  </span>
</div>

<style>
  .checker {
    display: flex;
    gap: 8px;
    align-items: center;
    text-align: center;
    font-size: 11px;
    margin: 8px 16px;
  }
  .box {
    width: 14px;
    height: 14px;
    background: #ffffff4d;
    border-radius: 4px;
    cursor: pointer;
    user-select: none;
  }
  span {
    font-size: 12px;
    cursor: pointer;
    user-select: none;
  }
  .highlight {
    background: #028AFB;
  }
  .freeze {
    opacity: .6;
    pointer-events: none;
  }
</style>
