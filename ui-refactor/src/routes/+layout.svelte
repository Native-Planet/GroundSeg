<div>
	<slot />
</div>

<style>
  @font-face {
    font-family: Inter;
    src: url("/Inter-SemiBold.otf");
  }
  div {
    font-family:Inter;
    background: url("/background.png") no-repeat center center fixed;
    -webkit-background-size: contain;
    -moz-background-size: contain;
    -o-background-size: contain;
    background-size: contain;
    background-color: #040404;
    height: 100vh;
    width: 100vw;
  }
</style>
