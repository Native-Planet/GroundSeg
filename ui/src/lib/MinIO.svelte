<script>
  import { api } from '$lib/api'
  import Fa from 'svelte-fa'
  import { faRotateRight } from '@fortawesome/free-solid-svg-icons'

  export let minio
  let loading = false

  const restartMinIO = () => {
    loading = true
    let module = 'minio'
	  fetch($api + '/system?module=' + module, {
			method: 'POST',
      credentials: 'include',
			headers: {'Content-Type': 'application/json'},
			body: JSON.stringify({'action':'reload'})
	  })
      .then(d => d.json())
      .then(res => {
        if (res === 200) {
          loading = false
          console.log("minio refreshed")
        }})
  }

</script>

<div class="minio">
  <div class="title-wrapper">
    <div class="title" class:disabled={!minio}>MinIO</div>
    <div
      on:click={restartMinIO}
      class:disabled={!minio}
      class:loading={loading}
      class="switch">
      <Fa icon={faRotateRight} size="1x" />
    </div>
  </div>
</div>

<style>
  @keyframes spin {
    from {transform: rotate(0deg);} 
    to {transform: rotate(360deg);}
  }
  .minio {
    background: #0000001d;
    padding: 20px 30px;
    border-radius: 8px;
    font-size: 18px;
    gap: 12px;
  }
  .title-wrapper {
    display: flex;
    align-items: center;
  }
  .title {
    font-size: 18px;
    flex: 1;
  }
  .switch {
    border-radius: 8px;
    padding: 2px;
    cursor: pointer;
    color: lime;
  }
  .loading {
    color: orange;
    animation-name: spin;
    animation-duration: 300ms;
    animation-iteration-count: infinite;
    animation-timing-function: linear;
  }
  .disabled {
    color: #ffffff;
    opacity: .2;
    pointer-events: none;
  }
</style>
