<script>
  import { api } from '$lib/api'
  import Fa from 'svelte-fa'
  import { faRotateRight } from '@fortawesome/free-solid-svg-icons/index.es'

  export let info
  let loading = false

  const restartMinIO = () => {
    loading = true
    const f = new FormData()
    const u = api + "/settings/minio"
    f.append('refresh', 'refresh')
    fetch(u, {method: 'POST',body: f})
      .then(d => d.json())
      .then(res => {
        if (res === 200) {
          loading = false
          console.log("minio refreshed")
    }})}

</script>

<div class="minio">
  <div class="title-wrapper">
    <div class="title">MinIO</div>
      {#if info}
        <div
          on:click={restartMinIO}
          class:disabled={!info.minio}
          class:loading={loading}
          class="switch">
          <Fa icon={faRotateRight} size="1x" />
        </div>
      {:else}
        <div class="blurred"></div>
      {/if}
    </div>
  </div>

<style>
  @keyframes spin {
    from {transform: rotate(0deg);} 
    to {transform: rotate(360deg);}
  }
  @keyframes breathe {
    0% {opacity: .6}
    50% {opacity: 0}
    100% {opacity: .6}
  }
  .minio {
    background: #0000006d;
    width: 300px;
    padding: 20px 40px 20px 40px;
    border-radius: 15px;
    font-size: 18px;
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
  .blurred {
    height: 25px;
    background: red;
    width: 25px;
    animation: breathe 2s infinite;
    background: #ffffff4d;
    filter: blur(6px);
    border-radius: 8px;
  }
</style>
