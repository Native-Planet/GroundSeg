<script>
  import { structure } from '$lib/stores/websocket'
  import Fa from 'svelte-fa'
  import { faArrowRight } from '@fortawesome/free-solid-svg-icons';
</script>

<div class="container">
  <div class="title">STARTRAM</div>
  <div class="body">
    <div class="status">status: running/stopped/unregistered</div>
    <div class="region">Active Region: Asia</div>
    <div class="expiry">Expiry Date: Date</div>
    <div class="region">Auto Renew: True</div>
    <button>Register Key</button>
    <button>Reregister Key</button>
    <button>Restart StarTram</button>
    <input type="text" />
    <button>Change Endpoint</button>
    <input type="password" />
    <button>Cancel Subscrition</button>
  </div>
</div>

<style>
  .container {
    margin: auto;
    width: calc(1104px - 80px);
    max-width: 98vw;
    padding: 40px;
  }
  .title {
    margin-bottom: 24px;
  }
</style>
