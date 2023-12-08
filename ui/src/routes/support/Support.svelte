<script>
  import { structure } from '$lib/stores/data'
</script>

<div class="container">
    <div class="title">CONTACT</div>
    <div class="spacer"></div>
    <div class="links">
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
    gap: 24px;
  }
  .spacer {
    flex: 1;
  }
  .link {
    text-decoration: none;
    color: var(--text-color);
    font-size: 16px;
  }
  .power-btn {
    background: var(--btn-secondary);
    color: var(--text-card-color);
    height: 42px;
    font-size: 14px;
    width: 200px;
    border-radius: 16px;
    cursor: pointer;
  }
</style>
