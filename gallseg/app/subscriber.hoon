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
++  on-save   on-save:def
++  on-load   on-load:def
++  on-watch  on-watch:def
++  on-leave  on-leave:def
++  on-peek   on-peek:def
++  on-arvo   on-arvo:def
++  on-fail   on-fail:def
::
++  on-init   
  ^-  (quip card _this)
  `this
::
++  on-poke
  |=  [=mark =vase]
  ^-  (quip card _this)
  |^
  ?>  =(src.bowl our.bowl)
  ?+    mark  (on-poke:def mark vase)
      %sub
    =^  cards  state
      (handle-poke !<(subscribe vase))
    [cards this]
  ==
  ::
  +$  subscribe  ?
  ++  handle-poke
    |=  =subscribe
    ^-  (quip card _state)
    :_  state
    :~  
      ?:  subscribe
        ~&  >  'sub called'
        [%pass /receive %agent [our.bowl %groundseg] %watch /broadcast]
      ~&  >  'unsub called'
      [%pass /receive %agent [our.bowl %groundseg] %leave ~]
    ==
  --
::
++  on-agent  ::  on-agent:def
  |=  [=wire =sign:agent:gall]
  ^-  (quip card _this)
  ?+    wire  (on-agent:def wire sign)
      [%receive ~]
    ?+    -.sign  (on-agent:def wire sign)
        %watch-ack
      ?~  p.sign
        ((slog '%todo-watcher: Subscribe succeeded!' ~) `this)
      ((slog '%todo-watcher: Subscribe failed!' ~) `this)
    ::
        %kick
      %-  (slog '%todo-watcher: Got kick, resubscribing...' ~)
      `this
      :::_  this
      :::~  [%pass /receive %agent [our.bowl %groundseg] %watch /broadcast]
      ::==
    ::
        %fact
      ?+    p.cage.sign  (on-agent:def wire sign)
          %broadcast
        ~&  >>  !<(broadcast q.cage.sign)
        `this
      ==
    ==
  ==
--
