<script>
  import { wide } from '$lib/stores/display';
  import { goto } from '$app/navigation';

  import Option from './Option.svelte';

</script>

<div class="card-wrapper {wide ? "wide" : "slim"}">
  <div class="title">LOAD A SHIP</div>
    <Option type="new" />
    <Option type="existing"/>
</div>

<style>
  .card-wrapper {
    background: var(--bg-base);
    border-radius: 16px;
    margin: auto;
    height: 70vh;
    padding: 40px;
    display:flex;
    flex-direction: column;
    align-items: center;
    display: flex;
    width: calc(1104px - 80px);
    flex-direction: column;
    align-items: center;
    gap: 40px;
  }
  .title {
    text-align: center;
    font-family: var(--title-font);
    color: #000;
    leading-trim: both;
    text-edge: cap;
    font-family: BPdotsUnicase;
    font-size: 48px;
    font-weight: 700;
    line-height: 64px; /* 133.333% */
    letter-spacing: -2.88px;
    text-transform: uppercase;
    margin-bottom: 0;
  }
  .selection {
  }
</style>
