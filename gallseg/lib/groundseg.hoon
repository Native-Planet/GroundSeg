/-  gs=groundseg
/+  a=admin
|%
::
++  reconnect
  |=  [now=@da interval=interval:gs]
  ^-  card:agent:gall
  =+  interval=~s10  ::  add exponential back off or something similar
  [%pass /reconnect %arvo %b %wait (add interval now)]
::
++  reconnect-cards
  |=  [our=@p now=@da state=state-0:gs]
  |^  ^-  (list card:agent:gall)
  (filter ~[do-connect set-timer])
  ::
  ++  do-connect
    |-  ^-  (unit card:agent:gall)
    ?.  =(%inactive session.state)  ~
    [~ (connect:a our)] 
  ::
  ++  set-timer
    |-  ^-  (unit card:agent:gall)
    ?.  =(%allow retry.policy.state)  ~
    [~ (reconnect now interval.policy.state)]
  ::
  ++  remove-null
    |=  a=(unit card:agent:gall)
    ^-  ?  ?~(a | &)
  ::
  ++  filter
    |=  v=(list (unit card:agent:gall))
    ^-  (list card:agent:gall)
    (turn (skim v remove-null) need)
  ::
  --
::
--
