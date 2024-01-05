<script>
  import { structure } from '$lib/stores/data'
  import { setStartramReminder, setAllStartramReminder } from '$lib/stores/websocket'
  $: urbits = $structure?.urbits || {}
  $: urbitKeys = Object.keys(urbits)
  $: selectAllCheck = () => {
    let reminderStatus = [];
    for (let patp in urbits) {
      if (!urbits[patp].info.startramReminder) {
        return false
      }
    }
    return true
  }

  const handleCheck = (patp, selected) => {
    setStartramReminder(patp, !selected)
  }

  const handleCheckAll = () => {
    setAllStartramReminder(!selectAllCheck)
  }

</script>
{#if urbitKeys.length < 1}
  <div class="no-ships-text">No available ships found on this device.</div>
{:else}
  <div class="ships">
    <div class="option">
      <div class="checkbox" on:click={handleCheckAll}>
        {#if selectAllCheck}
          <img class="checkmark" src="/checkmark.svg" alt="checkmark"  />
        {/if}
      </div>
      <div class="patp" on:click={handleCheckAll}>{selectAllCheck ? "Unselect" : "Select"} All</div>
    </div>
    {#each urbitKeys as patp}
      <div class="option">
        <div class="checkbox" on:click={()=>handleCheck(patp, urbits[patp].info.startramReminder)}>
          {#if urbits[patp].info.startramReminder}
            <img class="checkmark" src="/checkmark.svg" alt="checkmark"  />
          {/if}
        </div>
        <div class="patp" on:click={()=>handleCheck(patp, urbits[patp].info.startramReminder)}>{patp}</div>
      </div>
    {/each}
  </div>
{/if}

<style>
  .no-ships-text {
    font-size: 20px;
    text-align: left;
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-style: normal;
    font-weight: 300;
    letter-spacing: -1.44px;
  }
  .ships {
    display: flex; 
    flex-wrap: wrap;
    gap: 32px;
  }
  .option {
    display: flex;
    gap: 16px; 
    width: 380px;
    flex-grow: 1;
  }
  .checkbox {
    width: 24px;
    height: 24px;
    border: solid 2px var(--text-color);
    border-radius: 8px;
    background: var(--bg-modal);
    cursor: pointer;
  }
  .patp {
    cursor: pointer;
    font-size: 24px;
    text-align: left;
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-style: normal;
    letter-spacing: -1.44px;
  }
  .checkmark {
    height: 18px;
    width: 18px;
    padding: 3px;
  }
</style>
