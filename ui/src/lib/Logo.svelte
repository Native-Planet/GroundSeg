<script>
  export let t = ""
</script>
<div class="head">
  <a href="/"><img class="logo" src="/nplogo.svg" alt="native planet logo"/></a>
  <div class="title">{t}</div>
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
  height: 48px;
  width: auto;
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
