<script>
	import { page } from '$app/stores'
  import Header from './header.svelte'
  import Body from './body.svelte'
  import DeleteModal from './deletemodal.svelte'
  import Sigil from './sigil.svelte'
  $: patp = $page.params.patp
</script>

<div class="wrapper">
  <div class="mask"></div>
  <div class="sigil"><Sigil name={patp} /></div>
  <Header {patp} />
  <Body {patp} />
</div>

<style>
  .wrapper {
    position: relative;
    height: 743px;
    width: 1173px;
    margin: auto;
  }
  /* Sigil and Masks */
  .sigil {
    background-color: var(--fg-card);
    position: absolute;
    top: 16px;
    left: 16px;
    height: 134px;
    width: 134px;
    border-radius: 16px 0 16px 0;
  }
  .mask {
    background-color: var(--bg-card);
    position: absolute;
    height: 30px;
    width: 30px;
    top: 130px;
    left: 130px;
  }
</style>
