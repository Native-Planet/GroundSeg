/-  *groundseg, http-utils
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
++  roller-bind
  ^-  card
  [%pass /eyre/bind %arvo %e %connect [~ /~groundseg/roller] dap.bowl]
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
  |=  $:  [for=@ta secure=?]
          =request:http
      ==
  ^-  card
  =.  url.request  (roller-target header-list.request)
  =.  header-list.request
    %+  skip  header-list.request
    |=([k=@t @t] ?|  =('cookie' k)
                       =('x-groundseg-roller-url' k)
                   ==)
  =.  header-list.request
    =-  (set-header:http 'forwarded' - header-list.request)
    %+  rap  3
    :~  'for="groundseg";'
        'proto='  ?:(secure 'https' 'http')
    ==
  [%pass /roller-fetch/[for]/(scot %t url.request) %arvo %i %request request *outbound-config:iris]
::
++  on-init
  ^-  (quip card _this)
  :_  this
  ~[roller-bind]
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
  ~[roller-bind]
::
++  on-poke
  |=  [=mark =vase]
  ^-  (quip card _this)
  |^
  ?>  =(src.bowl our.bowl)
  ?+    mark  (on-poke:def mark vase)
      %handle-http-request
    =+  !<(order:http vase)
    =+  (purse:http url.request)
    ?.  ?=([%~groundseg %roller *] site)
      :_  this
      %^  spout:http  id
        [404 ~]
      `(as-octs:mimes:html (cat 3 'bad route into ' dap.bowl))
    [[(fetch-roller [id secure] request)]~ this]
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
  --
++  on-watch  ::  on-watch:def
  |=  =path
  ^-  (quip card _this)
  ?+    path  (on-watch:def path)
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
  ?+  wire
    ?.  ?=([%lick %soak *] sign)  (on-arvo:def +<)
    ?+    [mark noun]:sign        (on-arvo:def +<)
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
      [%eyre %bind ~]
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
      %+  spout:http  eid
      :-  [502 'x-groundseg-roller'^'cancelled' ~]
      ~
    ?>  ?=(%finished -.res)
    :_  this
    %+  spout:http  eid
    :-  =,  response-header.res
        :-  status-code
        (snoc headers 'x-groundseg-roller'^'finished')
    ?~  full-file.res  ~
    `data.u.full-file.res
  ==
::
++  on-fail   on-fail:def
--
