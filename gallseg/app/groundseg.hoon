/-  gs=groundseg
/+  default-agent, dbug, lib=groundseg, m=macro
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
  =+  policy=policy.state.this
  =.  retry.policy       %allow 
  =.  limit.policy           10
  =.  interval.policy      ~s30
  =.  policy.state.this  policy
  =.  last-contact.policy   [~]
  :_  this
  (reconnect-cards:lib our.bowl now.bowl state.this)
::
++  on-save  !>(state)
::
++  on-load
  |=  old-state=vase
  ^-  (quip card _this)
  =/  old  !<(versioned-state:gs old-state)
  ?-    -.old
      %0  
    :_  this(state old)
    (reconnect-cards:lib our.bowl now.bowl old)

  ==
::
++  on-poke
  |=  [=mark =vase]
  ?>  =(our.bowl src.bowl)
  ^-  (quip card _this)
  ?+    mark  (on-poke:def mark vase)
    ::
    ::  configure the agent
    ::
      %admin
    =/  act  !<(admin:gs vase)
    ?-    -.act
      ::
        %valve
      ?:  =(%open +.act)
        [~[[%pass / %arvo %l %spin /api]] this]
      [~[[%pass / %arvo %l %shut /api]] this]
      ::
        %retry
      =.  retry.policy.state.this  +.act
      `this
      ::
        %interval
      =.  interval.policy.state.this  +.act
      `this
      ::
        %limit
      =.  limit.policy.state.this  +.act
      `this
      ::
    ==
    ::
    ::  high level mark to interact with groundseg
    ::
      %macro
    =/  act  !<(macro:gs vase)
    ?-    -.act
      ::
        %verify
      :_  this
      (verify:m bowl token.state.this)
      ::
        %login
      :_  this
      (login:m bowl token.state.this password.act)
      ::
    ==
    ::
    ::  low level mark. No reason to use this
    ::
      %raw
    =/  act  !<(raw:gs vase)
    ?-    -.act
        %send
      :_  this
      (send:m our.bowl action.act)
    ==
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
  ::
  ?+    -.sign-arvo  ~&  >>>  -.sign-arvo  `this
      %behn
    ?>  =(/reconnect wire)
    :_  this
    (reconnect-cards:lib our.bowl now.bowl state.this)
    ::
      %lick
    =+  gift=+.sign-arvo
    ?+  -.gift  ~&  >>>  gift  `this
        %soak
      =.  last-contact.policy.state.this  [~ now.bowl]
      ?+    mark.gift  ~&  >>>  mark.gift  `this
          %connect
        :_  this(session %active)
        (verify:m bowl token.state.this)
        ::
          %disconnect
        `this(session %inactive)
      ==
      ::
    ==
    ::
  ==
  ::
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
