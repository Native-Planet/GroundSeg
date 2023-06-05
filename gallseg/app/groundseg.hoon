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
  =.  status.session.state.this
    %inactive
  =.  unlocked.settings.state.this
    &
  =.  reconnect-interval.settings.state.this
    ~s30
  `this
::
++  on-save
  !>(state)
::
++  on-load
  |=  old-state=vase
  ^-  (quip card _this)
  =/  old  !<(versioned-state:gs old-state)
  ?-    -.old
      %0  
    :_  this(state old)
    ::  TODO:
    ::  start timer that checks and tries to establish connection with
    ::  groundseg
    :~
      [%pass /timers %arvo %b %wait (add ~m1 now.bowl)]
    ==
  ==
::
++  on-poke
  |=  [=mark =vase]
  ^-  (quip card _this)
  ?+    mark  (on-poke:def mark vase)
      %control
    =/  act  !<(control:gs vase)
    ?-    -.act
        %unlocked
      ::  TODO:
      ::  lock/unlock %earth pokes
      `this
        %interval
      ::  TODO:
      ::  set reconnect interval
      `this
    ==
      %earth
    =/  act  !<(earth:gs vase)
    ?-    -.act
        %broadcast
      ::  TODO
      ::  merge broadcast into structure
      ::  set session to %active
      ^-  (quip card _this)
      =.  status.session.state.this
        %active
      `this(broadcast +.act)
        %activity
      ::  TODO
      ::  remove id from pending
      ::  set session to %active
      =.  status.session.state.this
        %active
      `this
    ==
      %agent
    =/  act  !<(agent:gs vase)
    ::
    ?-    -.act
        %connect
      ~&  >  'connect'
      `this
        %action
      ::  send to put directory
      ::  :hood &drum-put [/action/json +.act]
      ::  eg. '{"id":id,"payload":{"category":"token","module":null,"action":null}'
      :_  this
      ::  card
      :~  :*  %pass 
            /put
            %agent
            [our.bowl %hood]
            :+  %poke
              %drum-put 
            !>
            :-  /action/json
            +.act
      ==  ==  
      ::
    ==
    ::
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
++  on-arvo
  |=  [=wire =sign-arvo]
  ^-  (quip card _this)
  ?+    wire  (on-arvo:def wire sign-arvo)
      [%timers ~]
    ?+    sign-arvo  (on-arvo:def wire sign-arvo)
        [%behn %wake *]
      ?~  error.sign-arvo
        ((slog 'Call timer' ~) `this)
      (on-arvo:def wire sign-arvo)
    ==
  ==
::
++  on-leave  on-leave:def
::
++  on-peek   on-peek:def
::
++  on-agent  on-agent:def
::
++  on-fail   on-fail:def
::
--
