/-  gs=groundseg
|%
::
++  connect
  |=  our=@p
  ^-  card:agent:gall
  =+  poke=[%poke %admin !>([%valve %open])]
  [%pass /admin %agent [our %groundseg] poke]
::
++  disconnect
  |=  our=@p
  ^-  card:agent:gall
  =+  poke=[%poke %admin !>([%valve %open])]
  [%pass /admin %agent [our %groundseg] poke]
::
--
