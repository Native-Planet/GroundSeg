<script>
  import { goto } from '$app/navigation';
</script>


<div class="block" on:click={()=>goto("/boot")}>
  <div class="main-wrapper">
    <!-- Masks -->
    <div class="mask"></div>
    <div class="sigil-mask up"></div>
    <div class="sigil-mask down"></div>
    <!-- Sigil -->
    <div class="sigil"></div>
    <!-- Upper -->
    <div class="status"></div>
    <!-- Lower -->
    <div class="body">
      <div class="info"></div>
      <div class="settings-wrapper"></div>
    </div>
  </div>
  <div class="new">+</div>
</div>

<style>
  .block {
    position: relative;
  }
  .block:hover {
    cursor: pointer;
  }
  .block:hover .main-wrapper {
    cursor: pointer;
    opacity: 1;
  }
  .main-wrapper {
    color: var(--text-card-color);
    width: 288px;
    height: 148px;
    opacity: .6;
  }
  .new {
    font-family: var(--title-font);
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    color: var(--text-card-color);
    font-size: 64px;
    text-align: center;
    line-height: 140px;
  }
  /* Upper */
  .status {
    background-color: var(--bg-card);
    display: flex;
    align-items: center;
    gap: 8px;
    position: absolute;
    right: 0;
    height: 50px;
    width: 228px;
    border-radius: 8px 8px 0 0;
  }
  .status > .light {
    height: 14px;
    width: 21px;
    border-radius: 4px 0 0 4px;
  }
  .status > .on {
    background: lime;
  }
  .status > .off {
    background: var(--btn-secondary);
  }
  .status > .text {
    font-size: 9px;
    flex: 1;
    text-align: right;
  }

  /* Lower */
  .body {
    background-color: var(--bg-card);
    position: absolute;
    height: calc(148px - 50);
    width: 100%;
    bottom:0;
    border-radius: 8px 0 24px 8px;
  }
  /* Sigil and Masks */
  .sigil {
    background-color: var(--btn-secondary);
    position: absolute;
    height: 46px;
    width: 56px;
    margin: 4px 0 0 4px;
    border-radius: 8px 0 8px 0;
  }
  .mask {
    background-color: var(--bg-card);
    position: absolute;
    height: 30px;
    width: 30px;
    top: 30px;
    left: 50px;

  }
  .sigil-mask {
    background-color: var(--btn-secondary);
    position: absolute;
    height: 30px;
    width: 30px;
  }
  .up {
    top: 4px;
    left: 50px;
  }
  .down {
    top: 50px;
    left: 4px;
  }
  /* Info */
  .info {
    display: flex;
    align-items: center;
    height: 67px;
  }
  .patp {
    flex: 1;
    font-size: 16px;
    margin-left: 20px;
    font-family: var(--title-font);
  }
  .ext {
    position: relative;
    background: var(--text-color);
    height: 32px;
    width: 48px;
    border-radius: 160px 0 0 160px;
  }
  .icon {
    position: absolute;
    top: 6px;
    right: 16px;
    transform: rotate(-45deg);
  }
  /* Settings */
  .settings-wrapper {
    display: flex;
    margin-left: 20px;
    height: 32px;
  }
  .settings-info {
    flex: 1;
  }
  .settings {
    font-family: var(--title-font);
    flex: 1;
    display: flex;
    gap: 30px;
    font-size: 14px;
    line-height: 14px;
  }
  .settings-text {
    width: 20px;
  }
  .settings-button {
    background-color: var(--btn-secondary);
    background-image: url('/settings.svg');
    background-repeat: no-repeat;
    background-position: center;
    background-size: 50% 50%;
    background-color: var(--btn-secondary);
    height: 32px;
    width: 48px;
    border-radius: 24px 0;
  }
</style>
