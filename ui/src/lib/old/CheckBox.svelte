<script>
  import { createEventDispatcher } from 'svelte'
  import Fa from 'svelte-fa'
  import { faCheck } from '@fortawesome/free-solid-svg-icons'

  export let name = ''
  export let all = false
  export let check = true
  export let submitting = false

  const dispatch = createEventDispatcher()

  let gray = false

  $: check = loading(check)

  const loading = c => {
    gray = false
    return c
  }

  const handleCheck = () => {
    gray = true
    dispatch('update', all ? !check : name)
  }
</script>

<!-- Checkbox -->
<div class="checker" class:freeze={submitting || gray}>
  <div class="box" class:highlight={check} on:click={handleCheck}>
    {#if check}
      <Fa icon={faCheck} size="1x"/>
    {/if}
  </div>
  <span on:click={handleCheck}>
    {!all ? name : check ? 'Unselect all' : 'Select all'}
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
