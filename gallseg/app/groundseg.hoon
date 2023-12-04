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
  `this
::  
++  on-save
  ^-  vase
  !>(state)
::
++  on-load
  |=  old-state=vase
  ^-  (quip card _this)
  =/  old  !<(versioned-state old-state)
  ?-  -.old
    %0  `this(state old)
  ==
::
++  on-poke
  |=  [=mark =vase]
  ^-  (quip card _this)
  |^
  ?>  =(src.bowl our.bowl)
  ?+    mark  (on-poke:def mark vase)
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
      ~&  >>>  'SHUT LICK PORT. TEMPORARILY DISABLED'
      ~ 
      :::~  [%pass /lick %arvo %l %shut /'groundseg.sock']
      ::==
    :_  this
    :~  [%give %fact ~[/broadcast] %broadcast !>(`broadcast`noun.sign)]
    ==
  ==
::
++  on-fail   on-fail:def
--
