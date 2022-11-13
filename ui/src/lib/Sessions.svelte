<script>
  import { api } from '$lib/api'
  import PrimaryButton from '$lib/PrimaryButton.svelte'

  export let sessions

  let loading = false
  let loadingAll = false

  const logoutSessions = () => {
    loading = true
    let module = 'session'
	  fetch($api + '/system?module=' + module, {
			method: 'POST',
      credentials: 'include',
			headers: {'Content-Type': 'application/json'},
			body: JSON.stringify({'action':'logout'})
	  })
      .then(d => d.json())
      .then(res => {
        if (res === 200) {
          loading = false
          window.location.href = "/login"
        }
        console.log(res)
      })
  }

  const logoutAll = () => {
    loadingAll = true
    let module = 'session'
	  fetch($api + '/system?module=' + module, {
			method: 'POST',
      credentials: 'include',
			headers: {'Content-Type': 'application/json'},
			body: JSON.stringify({'action':'logout-all'})
	  })
      .then(d => d.json())
      .then(res => {
        if (res === 200) {
          loadingAll = false
          window.location.href = "/login"
        }
        console.log(res)
      })
  }

</script>

<div class="minio">
  <div class="title-wrapper">
    <div class="title">Security</div>
  </div>
    <div class="button-wrapper">
      <PrimaryButton 
        background="black"
        standard="Logout"
        loading="Logging out..."
        status={loading ? "loading" : "standard"}
        on:click={logoutSessions}
      />
    {#if sessions > 1}
      <PrimaryButton 
        left={false}
        background="#ffffff4d"
        standard="Logout everywhere"
        loading="Logging out of all sessions"
        status={loadingAll ? "loading" : "standard"}
        on:click={logoutAll}
      />
    {/if}
    </div>
</div>

<style>
  .minio {
    background: #0404044d;
    padding: 20px 40px 20px 40px;
    border-radius: 15px;
    font-size: 18px;
  }
  .title-wrapper {
    display: flex;
    align-items: center;
    margin-bottom: 24px;
  }
  .title {
    font-size: 18px;
    flex: 1;
  }
  .button-wrapper {
    display: flex;
    gap: 12px;
    align-items: end;
  }
</style>
