/-  *groundseg
/+  default-agent, dbug
|%
+$  versioned-state
  $%  state-0
  ==
+$  state-0  [%0 =alive]
+$  card  card:agent:gall
--
=|  state-0
=*  state  -
%-  agent:dbug
^-  agent:gall
|_  =bowl:gall
+*  this  .
    def   ~(. (default-agent this %.n) bowl)
::
++  on-init
  ^-  (quip card _this)
  :_  this
  ~[[%pass /eyre/connect %arvo %e %connect [~ /'~groundseg'/roller] dap.bowl]]
::  
++  on-save
  ^-  vase
  !>(state)
::
++  on-load
  |=  old-state=vase
  ^-  (quip card _this)
  =/  old=state-0
    =/  loaded  (mule |.(!<(state-0 old-state)))
    ?:  ?=(%& -.loaded)  p.loaded
    *state-0
  :_  this(state old)
  ~[[%pass /eyre/connect %arvo %e %connect [~ /'~groundseg'/roller] dap.bowl]]
::
++  on-poke
  |=  [=mark =vase]
  ^-  (quip card _this)
  |^
  ?>  =(src.bowl our.bowl)
  ?+    mark  (on-poke:def mark vase)
      %handle-http-request
    (handle-http !<([@ta inbound-request:eyre] vase))
  ::  toggle lick port
      %port
    =^  cards  state
      (handle-port !<(? vase))
    [cards this]
   ::  spit cord
      %action
    =^  cards  state
      (handle-action !<(action vase))
    [cards this]
      %heartbeat
    =^  cards  state
      (handle-heartbeat !<(@ vase))
    [cards this]
  ==
  ::
  ++  handle-http
    |=  [eid=@ta req=inbound-request:eyre]
    ^-  (quip card _this)
    ?:  =(%'OPTIONS' method.request.req)
      :_  this
      %^  give-http  eid
        [204 ~[['access-control-allow-origin' '*'] ['access-control-allow-methods' 'POST, OPTIONS'] ['access-control-allow-headers' 'Content-Type, X-Groundseg-Roller-URL'] ['access-control-max-age' '3600']]]
      ~
    [[(fetch-roller eid req)]~ this]
  ::
  ++  handle-port
    |=  open=?
    ^-  (quip card _state)
    :_  state
    :~  
      ?:  open
        [%pass /lick %arvo %l %spin /'groundseg.sock']
      [%pass /lick %arvo %l %shut /'groundseg.sock']
    ==
  ::
  ++  handle-action
    |=  act=action
    ^-  (quip card _state)
    :_  state(alive now.bowl)
    ~[[%pass /lick %arvo %l %spit /'groundseg.sock' %action act]]
  ++  handle-heartbeat
    |=  b=@
    ^-  (quip card _state)
    `state(alive now.bowl)
  ::
  ++  roller-target
    |=  headers=(list [@t @t])
    ^-  @t
    =/  target  (get-header:http 'x-groundseg-roller-url' headers)
    ?~  target
      'https://roller.urbit.org/v1/roller'
    u.target
  ::
  ++  fetch-roller
    |=  [eid=@ta req=inbound-request:eyre]
    ^-  card
    =/  request=request:http  request.req
    =.  url.request  (roller-target header-list.request)
    =.  header-list.request
      %+  skip  header-list.request
      |=  [k=@t @t]
      ?|  =('cookie' k)
          =('x-groundseg-roller-url' k)
      ==
    =.  header-list.request
      =-  (set-header:http 'forwarded' - header-list.request)
      %+  rap  3
      :~  'for="groundseg";'
          'proto='  ?:(secure.req 'https' 'http')
      ==
    [%pass /roller-fetch/[eid]/(scot %t url.request) %arvo %i %request request *outbound-config:iris]
  ::
  ++  give-http
    |=  [eid=@ta =response-header:http data=(unit octs)]
    ^-  (list card)
    =/  =path  /http-response/[eid]
    :~  [%give %fact ~[path] %http-response-header !>(response-header)]
        [%give %fact ~[path] %http-response-data !>(data)]
        [%give %kick ~[path] ~]
    ==
  --
++  on-watch  ::  on-watch:def
  |=  =path
  ^-  (quip card _this)
  ?+    path  (on-watch:def path)
      [%http-response @ ~]
    `this
      [%broadcast ~]
    :_  this(alive now.bowl)
    :~  
      [%give %fact ~ %broadcast !>(`broadcast`'{"type":"init"}')]
      [%pass /lick %arvo %l %spin /'groundseg.sock']
    ==
  ==
::
++  on-leave  on-leave:def
++  on-peek   on-peek:def
++  on-agent  on-agent:def
::
++  on-arvo
  |=  [=wire sign=sign-arvo]
  ^-  (quip card _this)
  |^
  ?+  wire
    ?.  ?=([%lick %soak *] sign)  `this
    ?+    [mark noun]:sign        `this
        [%connect ~]
      ((slog 'groundseg socket connected' ~) `this)
      ::
        [%disconnect ~]
      ((slog 'groundseg socket disconnected' ~) `this)
      ::
        [%error *]
      ((slog leaf+"socket {(trip ;;(@t noun.sign))}" ~) `this)
      ::
        [%broadcast *]
      ?.  ?=(@ noun.sign)
        ((slog 'invalid broadcast' ~) `this)
      ?:  (gte `@dr`(sub now.bowl alive.state) ~s15)
        :_  this
        :~  [%pass /lick %arvo %l %shut /'groundseg.sock']
        ==
      :_  this
      :~  [%give %fact ~[/broadcast] %broadcast !>(`broadcast`noun.sign)]
      ==
    ==
      [%eyre %connect ~]
    ?>  ?=(%bound +<.sign)
    ?:  accepted.sign
      ((slog 'groundseg roller proxy bound' ~) `this)
    ((slog 'groundseg roller proxy bind failed' ~) `this)
      [%roller-fetch @ @ ~]
    =/  eid=@ta  i.t.wire
    ?>  ?=([%iris %http-response *] sign)
    =*  res  client-response.sign
    =?  res  ?=(%progress -.res)
      [%cancel ~]
    ?:  ?=(%cancel -.res)
      :_  this
      (give-status eid 502 'cancelled')
    ?>  ?=(%finished -.res)
    :_  this
    %^  give-http  eid
      [status-code.response-header.res (response-headers headers.response-header.res)]
    ?~  full-file.res  ~
    `data.u.full-file.res
  ==
  ::
  ++  give-http
    |=  [eid=@ta =response-header:http data=(unit octs)]
    ^-  (list card)
    =/  =path  /http-response/[eid]
    :~  [%give %fact ~[path] %http-response-header !>(response-header)]
        [%give %fact ~[path] %http-response-data !>(data)]
        [%give %kick ~[path] ~]
    ==
  ::
  ++  give-status
    |=  [eid=@ta status=@ud msg=@t]
    ^-  (list card)
    %^  give-http  eid
      [status ~[['content-type' 'text/plain']]]
    `(as-octs:mimes:html msg)
  ::
  ++  response-headers
    |=  headers=(list [@t @t])
    ^-  (list [@t @t])
    =/  clean=(list [@t @t])
      %+  skip  headers
      |=  [key=@t value=@t]
      ?|  =(key 'transfer-encoding')
          =(key 'connection')
      ==
    %+  weld  clean
    ~[['x-groundseg-roller' 'finished'] ['access-control-allow-origin' '*']]
  --
::
++  on-fail   on-fail:def
--
