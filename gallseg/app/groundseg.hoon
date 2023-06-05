/-  gs=groundseg
/+  default-agent, dbug
|%
+$  card  card:agent:gall
--
%-  agent:dbug
=|  state-0:gs
=*  state  -
^-  agent:gall
|_  =bowl:gall
+*  this  .
  def   ~(. (default-agent this %.n) bowl)
::
++  on-init
  ^-  (quip card _this)
  `this(session %inactive, unlocked &)
::
++  on-save
  !>(state)
::
++  on-load
  |=  old-state=vase
  ^-  (quip card _this)
  =/  old  !<(versioned-state:gs old-state)
  ?-  -.old
        %0  
    :_  this(state old)
    ::  TODO:
    ::  start timer that checks and tries to establish connection with
    ::  groundseg
    ~
  ==
::
++  on-poke
  |=  [=mark =vase]
  ^-  (quip card _this)
  ?+    mark  (on-poke:def mark vase)
      %control
    =/  act  !<(control:gs vase)
    ?-    -.act
        %access
          `this(unlocked +.act)
        %
      %earth
    =/  act  !<(earth:gs vase)
    ?-    -.act
        %broadcast
      ::  TODO
      ::  merge broadcast into structure
      `this(broadcast +.act, session %active)
        %activity
      ::  TODO
      ::  remove id from pending
      ::  set session to %active
      `this(session %active)
    ==
      %mars
    =/  act  !<(mars:gs vase)
    :_  this
    ~
    ::  send to put directory
    ::  :hood &drum-put [/action/json +.act]
    ::  eg. '{"id":id,"payload":{"category":"token","module":null,"action":null}'
  ==
::
++  on-watch
  ::  You are able to subscribe to:
  ::  1. entire broadcast
  ::  2. a category
  ::  3. a module
  ::  4. a ship
  ::  eg:
  ::  [%receive %system %startram %container ~]
  ::  [%receive ~]
  ::  [%receive %zod %container %status ~]
  on-watch:def
::  |=  =path
::  ^-  (quip card _this)
::  ?+    path  (on-watch:def path)
::      [%receive ~]
::    ?:  =(session %inactive)
::
::    ?>  =(our.bowl src.bowl)
::    :_  this
::    :~  [%give %fact ~ %todo-update !>(`update:todo`initial+tasks)]
::==
::==
::
++  on-leave  on-leave:def
::
++  on-peek   on-peek:def
::
++  on-agent  on-agent:def
::
++  on-arvo   on-arvo:def
::
++  on-fail   on-fail:def
::
--
