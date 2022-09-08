<script>
  import { api } from '$lib/api'
  import PrimaryButton from '$lib/PrimaryButton.svelte'
  import LinkButton from '$lib/LinkButton.svelte'

  export let name='', key=''

  let buttonStatus = 'standard'

  const boot = () => {
    buttonStatus = 'loading'
    const f = new FormData()
    const u = api + "/upload/key"
    f.append("patp", name)
    f.append("key", key)
    fetch(u, {method: 'POST',body: f})
      .then(d => d.json())
      .then(res => {
        if (res === 200) {
          buttonStatus = 'success'
          setTimeout(window.location.href = "/" + name, 2000)
        }
      })
  }
</script>

<div>

  {#if buttonStatus != 'loading'}
    <LinkButton
      text="Cancel"
      src="/"
      disabled={false}
    />
  {/if}

  <PrimaryButton
    on:click={boot}
    standard="Create new pier"
    success="Pier created. Redirecting..."
    failure="Failed to create pier"
    loading="Your pier is being created.."
    status={(name == '') || (key == '') ? "disabled" : buttonStatus}
    left={false}
  />

</div>

<style>
  div {
    display: flex;
    margin-top: 24px;
  }
</style>
