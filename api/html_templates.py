head = """\n
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>GroundSeg Connect to Connect</title>
    <style>
        @font-face {
          font-family: Inter;
            src: url('/static/Inter-SemiBold.otf')
        }
        body {
          font-family: Inter;
          margin: 0;
          width: 100vw;
          height: 100vh;
          /*
          background: url("/static/background.png") no-repeat center center fixed;
          -webkit-background-size: contain;
          -moz-background-size: contain;
          -o-background-size: contain;
          background-size: contain;
          background-color: #040404;
          */
        }
        .card::-webkit-scrollbar {display: none;}
        .card {
          font-family: inherit;
          position: fixed;
          top: 50%;
          left: 50%;
          transform: translate(-50%, -50%);
          color: #fff;
          
          border-radius: 20px;

          background: #6d6d6d33;
          backdrop-filter: blur(60px);
          -moz-backdrop-filter: blur(60px);
          -o-backdrop-filter: blur(60px);
          -webkit-backdrop-filter: blur(60px);

          -ms-overflow-style: none;
          scrollbar-width: none;

          overflow: scroll;
          padding-bottom: 20px;
          min-width: 400px;
          max-width: 100vw;
          max-height: 80vh;
        }
        .logo {
          padding: 20px;
        }
        img {
          height: 32px;
          float: left;
        }
        .text {
          font-size: 14px;
          padding-left: 18px;
          line-height: 32px;
        }
        .title {
          font-size: 14px;
          font-weight: 700;
          padding-bottom: 12px;
          text-align: center;
        }
        a.ssid {
          display: block;
          font-family: inherit;
          font-size: 13px;
          width: 400px;
          -webkit-appearance: button;
          -moz-appearance: button;
          appearance: button;
          text-decoration: none;
          text-align: left;
          border: none;
          background: none;
          color: white;
          padding: 16px 0 16px 20px;
        }
        .ssid + .ssid:before {
          border-top: solid 1px #ffffff4d;
        }
        a.back {
          -webkit-appearance: button;
          -moz-appearance: button;
          appearance: button;
          text-decoration: none;
          background: #ffffff4d;
          color: white;
          font-family: inherit;
          border-radius: 6px;
          border: none;
          padding: 8px 0 8px 0;
          width: 80px;
          font-size: 12px;
          text-align: center;
          margin-left: 20px;
        }
        a:hover {
          cursor: pointer;
        }
        a.ssid:hover {
          background: #0404044d;
        }
        form {
          display: inline;
        }
        button.rescan {
          float: right;
          background: #ffffff4d;
          color: white;
          font-family: inherit;
          border-radius: 6px;
          border: none;
          width: 80px;
          padding: 8px;
          font-size: 12px;
        }
        button.connect {
          display: block;
          float: right;
          background: #008EFF;
          color: white;
          font-family: inherit;
          border-radius: 6px;
          border: none;
          padding: 8px;
          width: 80px;
          margin-right: 20px;
          font-size: 12px
        }
        input {
          font-family: inherit;
          color: white;
          display: block;
          text-align: center;
          width: 400px;
          margin: 0 20px 20px 20px;
          padding: 8px;
          font-size: 12px;
          border: none;
          border-radius: 8px;
          background: #ffffff4d;
        }
        input::placeholder {
          color: white;
        }
        input:focus {
          outline: none;
        }
        button:hover {
          cursor: pointer;
        }
        .sep {
          height: 0;
          width: 100%;
          border-bottom: solid 1px #ffffff4d;
        }
    </style>
</head>
  """

def home_page(ssids):
    formatted_ssids = ''.join(list(map(lambda z: f'<div><a href="/connect/{z}">{z}</a></div>', ssids)))

    body = f"""\n
{head}
<body>
  <form action="/" method="post">
    <button type="submit">Restart</button>
  </form>
  <div>CONNECT TO NETWORK</div>
  <div>Choose a Wi-Fi network</div>
  <!-- List of SSIDs -->
  {formatted_ssids}
</body>
"""

    return body

def connect_page(ssid):

    body = f"""\n
{head}
<body>
  <!-- Header -->
  <form action="/" method="post">
    <button type="submit">Restart</button>
  </form>
  <div>{ssid}</div>
  <form method="post">
    <input type="text" placeholder="Password for {ssid}" name="password" />
    <a href="/">Back</a>
    <button type="submit">Connect</button>
  </form>
</body>
"""
    
    return body
