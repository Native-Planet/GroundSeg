<script>
  import { structure } from '$lib/stores/websocket'
  import { page } from '$app/stores'
</script>

<div class="container">
  <div class="title">SUPPORT</div>
  <div class="spacer"></div>
  <div class="links">
    <a href="http://{$page.url.hostname}:19999" target="_blank" class="link">Netdata</a>
    <a href="https://twitter.com/NativePlanetIO" target="_blank" class="link">Twitter</a>
    <a href="https://twitter.com/NativePlanetIO" target="_blank" class="link">Email</a>
    <a href="https://twitter.com/NativePlanetIO" target="_blank" class="link">Urbit</a>
  </div>
</div>

<style>
  .container {
    margin: 0;
    display: flex;
    align-items: center;
    padding: 20px 40px;
    margin-bottom: 20px;
  }
  .title {
    margin-bottom: 0;
  }
  .links {
    display: flex;
    gap: 20px;
    font-weight: 600;
  }
  .spacer {
    flex: 1;
  }
  .link {
    text-decoration: none;
    color: var(--text-color);
    font-size: 12px;
  }
</style>
