<script>
  export let on
  const handleToggle = () => console.log("toggle")
</script>
<div class="wrapper" class:on={on}>
  <div class="title">StarTram</div>
  <div on:click={handleToggle} class="outer">
    <div class="inner" style="margin-left:{on ? 20 : 4}px"></div>
  </div>
</div>
<style>
  .on > .title {
    color: var(--text-card-color);
  }
  .on > .outer {
    background: #077D13;
  }
  .on > .outer > .inner {
    background: #07D91C;
  }
  .wrapper {
    display: flex;
    gap: 5px;
    align-items: center;
    margin-right: 3px;
  }
  .title {
    color: #637061;
    text-align: right;
    leading-trim: both;
    text-edge: cap;
    font-family: var(--title-font);
    font-size: 14px;
    font-style: normal;
    font-weight: 700;
    line-height: normal;
    letter-spacing: -0.42px;
    text-transform: uppercase;
    margin-bottom: 2px;
  }
  .outer {
    display: inline-flex;
    width: 40px;
    height: 24px;
    align-items: center;
    flex-shrink: 0;

    border-radius: 4px 0px 0px 4px;
    background: var(--text-color, #313933);
  }
  .inner {
    width: 16px;
    height: 16px;
    border-radius: 4px;
    background: #161D17;
    transition: margin-left 0.2s ease;
  }
</style>
