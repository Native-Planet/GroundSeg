/-  *groundseg
/+  default-agent, dbug
|%
+$  versioned-state
  $%  state-0
  ==
+$  state-0  [%0 @ud]
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
++  on-save   on-save:def
::
++  on-load   on-load:def
::
++  on-poke
  |=  [=mark =vase]
  ^-  (quip card _this)
  |^
  ?>  =(src.bowl our.bowl)
  ?+    mark  (on-poke:def mark vase)
      %port
    =^  cards  state
      (handle-poke !<(power vase))
    [cards this]
      %action
      =^  cards  state
      (handle-action !<(action vase))
    [cards this]
  ==
  ::
  +$  power  ?(%on %off)
  ++  handle-poke
    |=  p=power
    ^-  (quip card _state)
    ?-    p
        %on
      :_  state
      :~  [%pass /lick %arvo %l %spin /'groundseg.sock']
      ==
    ::
        %off
      :_  state
      :~  [%pass /lick %arvo %l %shut /'groundseg.sock']
      ==
    ==
  ++  handle-action
    |=  act=action
    ^-  (quip card _state)
    ~&  >  'handle action'
    :: check if port is running
    :: send to port
    `state
  --
++  on-watch  ::  on-watch:def
  |=  =path
  ^-  (quip card _this)
  ?+    path  (on-watch:def path)
      [%broadcast ~]
    :_  this
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
    ((slog 'socket connected' ~) `this)
    ::
      [%disconnect ~]
    ((slog 'socket disconnected' ~) `this)
    ::
      [%error *]
    ((slog leaf+"socket {(trip ;;(@t noun.sign))}" ~) `this)
    ::
      [%broadcast *]
    ?.  ?=(@ noun.sign)
      ((slog 'invalid broadcast' ~) `this)
    %-  (slog 'message' ~)
    :_  this
    ~&  >  `broadcast`noun.sign
    :~  [%give %fact ~[/broadcast] %broadcast !>(`broadcast`noun.sign)]
    ==
  ==
::
++  on-fail   on-fail:def
--
