<script>
  export let t = ""
  export let back = false
</script>
<div class="head">
  <a href="/"><img class="logo" src="/nplogo.svg" alt="native planet logo"/></a>
  <div class="title">{t}</div>
  {#if back}
    <a class="right" href="/"><img class="home" src="/home.svg" alt="home button" /></a>
  {/if}
</div>
<style>
  .head {
    display: flex;
    align-items: start;
  }

 img {
   margin: 20px 0 0 20px;
 }
 .logo {
  width: 48px;
 }
  .title {
    font-size: 18px;
    padding-left: 18px;
    margin-top: auto;
    padding-bottom: 8px;
  }
  .right {
    margin-left: auto;
    padding-right: 20px;
  }
  .home {
    width: 32px;
  }

</style>
