<script>
  import { structure } from '$lib/stores/data'
  import { page } from '$app/stores'
  import { openModal } from 'svelte-modals'
  import BugReportModal from './bugreportmodal.svelte'
</script>

<div class="container">
  <div class="sys-title">SUPPORT</div>
  <div class="spacer"></div>
  <div class="links">
    <button
      on:click={()=>openModal(BugReportModal)}
      class="link">
      Report Bug
    </button>
    <a href="http://{$page.url.hostname}:19999" target="_blank" class="link">Netdata</a>
    <span>|</span>
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
    padding-top: 32px;
    padding-bottom: 32px;
    margin-bottom: 20px;
  }
  .sys-title {
    margin-bottom: 0;
  }
  .links {
    display: flex;
    gap: 24px;
    font-weight: 600;
  }
  .spacer {
    flex: 1;
  }
  .link {
    text-decoration: none;
    color: var(--NP_Black, #313933);
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    line-height: 56px; /* 233.333% */
    letter-spacing: -1.44px;
    cursor: pointer;
  }
  .link:hover {
    text-decoration: underline;
  }
  span {
    color: var(--Gray-200, #ABBAAE);
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    line-height: 56px; /* 233.333% */
    letter-spacing: -1.44px;
  }
</style>
