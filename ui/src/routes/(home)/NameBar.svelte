<script>
  export let patp
  export let running
</script>

<div class="outer">
  <div class="status" class:running={running}>
    {running ? "online" : "off"}
  </div>
  <div class="patp">{patp}</div>
</div>

<style>
  .status {
    color: var(--Gray-300, #8FA393);
    leading-trim: both;
    text-edge: cap;
    font-family: var(--title-font);
    font-size: 14px;
    font-style: normal;
    font-weight: 700;
    line-height: normal;
    letter-spacing: -0.42px;
    text-transform: uppercase;
  }
  .patp {
    color: var(--text-card-color); 
    leading-trim: both;
    text-edge: cap;
    font-family: var(--title-font);
    font-size: 24px;
    font-style: normal;
    font-weight: 700;
    line-height: 16px;
    letter-spacing: -0.72px;
    text-transform: uppercase;
  }
  .running {
    color: #07D91C;
  }
</style>
