<script>
  import { wide } from '$lib/stores/display'
  import './system.css'
  import Connection from './Connection.svelte'
  import SystemDetails from './SystemDetails.svelte'
  import Power from './Power.svelte'
  import Netdata from './Netdata.svelte'
  import Logs from './Logs.svelte'
  import Support from './Support.svelte'
</script>

<Connection {wide} />
<div class="spacer"></div>
<SystemDetails {wide} />
<div class="spacer"></div>
<Power {wide} />
<div class="spacer"></div>
<Netdata {wide} />
<div class="spacer"></div>
<Logs {wide} />
<div class="spacer"></div>
<Support {wide} />
<div class="spacer"></div>

<style>
  .spacer {
    height: 40px;
  }
</style>
