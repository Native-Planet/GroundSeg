<script>
  import { structure, URBIT_MODE, daysUntilDate, startramMaxReminderDays } from '$lib/stores/data'
  import { goto } from '$app/navigation';

  $: startram = $structure?.profile?.startram || {}
  $: registered = startram?.info?.registered || false
  $: running = startram?.info?.running || false
  $: fillColor = !registered ? "#00000000" : !running ? "red" : "lime"

  $: expiry = startram?.info?.expiry || "9999-12-31"
  $: pfx = $URBIT_MODE ? "/apps/groundseg" : ""

  $: daysLeft = daysUntilDate(expiry)

  const handleDaysLeft = () => {
    goto(pfx+"/profile#startram")
  }

</script>

<div class:disabled={!registered} on:click={()=>{goto(pfx+"/profile#startram")}}>
</div>

<style>
  div {
    width: 24px;
    height: 24px;
    cursor: pointer;
    margin-bottom: 4px;
  }
  .disabled {
    display: none;
  }
</style>
