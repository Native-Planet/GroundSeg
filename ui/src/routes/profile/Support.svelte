<script>
  import { structure } from '$lib/stores/data'
  export let wide
</script>

<div class="container {wide ? "wide" : "slim"}">
    <div class="title">Support</div>
    <div class="links">
      <a href="https://twitter.com/NativePlanetIO" target="_blank" class="link">Twitter</a>
      <a href="https://twitter.com/NativePlanetIO" target="_blank" class="link">Email</a>
      <a href="https://twitter.com/NativePlanetIO" target="_blank" class="link">Urbit</a>
    </div>
</div>

<style>
  .wide {
    width: 992px;
    max-width: 98vw;
    padding: 56px;
  }
  .slim {
    width: 100vw;
  }
  .title {
    margin-bottom: 24px;
  }
  .links {
    display: flex;
    gap: 20px;
  }
  .link {
    text-decoration: none;
    color: var(--text-color);
    font-size: 12px;
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
