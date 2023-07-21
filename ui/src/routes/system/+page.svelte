<script>
  import { wide } from '$lib/stores/display'
  import './system.css'
  import Connection from './Connection.svelte'
  import SystemDetails from './SystemDetails.svelte'
  import Power from './Power.svelte'
</script>

<Connection {wide} />
<div class="spacer"></div>
<SystemDetails {wide} />
<div class="spacer"></div>
<Power {wide} />
<div class="spacer"></div>
<Netdata />
<!--
<Logs />
<Support />
-->
<style>
  .spacer {
    height: 40px;
  }
</style>
