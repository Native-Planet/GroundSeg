/-  gs=groundseg
|%
::
++  make-id
  |=  =bowl:gall
  =,  bowl
  ^-  id:gs
  =+  hash=`@ux`(sham :(add eny now our))
  =+  text=(trip `@t`(scot %ux hash))
  (crip (scag 16 (slag 2 (skip text |=(t=@t =(t '.'))))))

++  send
  |=  [our=@p =action:gs]
  ^-  (list card:agent:gall)
  ~&  >  'macro:send action via %lick'
  :~  [%pass / %arvo %l %spit /api [%action !>(action)]]
  ==
::
++  action-build-send
  |=  [=bowl:gall =payload:gs =token:gs]
  (send our.bowl `action:gs`[%action (make-id bowl) payload token])

++  verify
  |=  [=bowl:gall =token:gs]
  ^-  (list card:agent:gall)
  =/  =payload:gs  [[~ %token] ~ ~ ~]
  (action-build-send bowl payload token)
::
++  login
  |=  [=bowl:gall =token:gs =password:gs]
  ^-  (list card:agent:gall)
  =/  =payload:gs
    [[~ 'system'] ~ [~ 'login'] [~ password]]
  (action-build-send bowl payload token)
::
--
