/-  gs=groundseg
/+  default-agent, dbug, lib=groundseg, m=macro
|%
+$  versioned-state
  $%  state-0
  ==
+$  state-0  [%0 =blob connected=?]
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
  :~  [%pass / %arvo %l %spin /'groundseg.sock']
  ==
::
++  on-save  !>(state)
::
++  on-load
  |=  old-vase=vase
  ^-  (quip card _this)
  [~ this(state !<(state-0 old-vase))]
::
++  on-poke
  |=  [=mark =vase]
  ?>  =(our.bowl src.bowl)
  ^-  (quip card _this)
  ?>  ?=(%penpai-do mark)
  ?>  =(our.bol src.bol)
  =+  !<(=do vase)
  ?-    -.do
      %post
  ?.  connected  !!
  :_  this
  :~  [%pass /spit %arvo %l %spit /'groundseg.sock' %json +.do]
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
  |=  [=wire sign=sign-arvo]
  ^-  (quip card _this)
  ?.  ?=([%lick %soak *] sign)  (on-arvo:def +<)
  ?+    mark.sign  (on-arvo:def +<)
      %connect     
    ~&  'socket connected'
    :-  ~
    this(connected %.y)
      %disconnect
    ~&  'socket disconnected'
    :-  ~
    this(connected %.n)
      %error       ((slog leaf+"socket {(trip ;;(@t noun.sign))}" ~) `this)
      %json
    =+  ;;(in-blob=blob noun.sign)
    :_  this(blob blob)
    :~  (fact:io groundseg-did+!>(`did`[%json in-blob]) /all ~)
    ==
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
