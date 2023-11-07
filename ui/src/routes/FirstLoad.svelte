<script>
</script>

<div class="bg">
  <img src="/nplogo.svg" alt="native planet" />
  <div class="text">LOADING</div>
</div>

<style>
  .bg {
    background: var(--bg-modal);
    width: 100vw;
    height: 100vh;
    display: flex;
    flex-direction: column;
    justify-content: center;
    align-items: center;
    gap: 80px;
  }
  img {
    height: 200px;
    filter: invert(100);
  }
  .text {
    font-family: var(--title-font);
    color: #000;
    leading-trim: both;
    text-edge: cap;
    font-size: 64px;
    font-style: normal;
    font-weight: 700;
    line-height: normal;
    letter-spacing: -1.92px;
    animation: breathe 5s infinite;
  }
  @keyframes breathe {
    0% {
      opacity: .2;
    }
    50% {
      opacity: 1;
    }
    100% {
      opacity: .2;
    }
  }
</style>
