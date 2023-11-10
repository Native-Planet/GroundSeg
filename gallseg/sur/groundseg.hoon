|%
::
::  Types
::
+$  state-0  [%0 =blob connected=? =policy =session]
+$  blob          @t
+$  do
  $%  [%json =blob]
  ==
+$  did
  $%  [%post =blob]
  ==
::
::  Old GS stuff
::
+$  id            @t
+$  session       ?(%active %inactive)
+$  last-contact  (unit @da)
+$  retry         ?(%allow %block)
+$  valve         ?(%open %shut)
+$  limit         @ud
+$  interval      @dr
+$  pending       (map id created=@da)
+$  token         (unit [i=id t=@t])
+$  policy        [=last-contact =retry =limit =interval]
+$  password      @t
::
::  GroundSeg
::
+$  groundseg  $%  [%action @t]
                   [%broadcast @t]
                   [%activity @t]
               ==
::
::  GroundSeg Broadcast
::
+$  broadcast     @t
::
::  GroundSeg Action
::
+$  action    [%action =id =payload =token]
+$  payload   [=category =uship =module =act]
+$  category  (unit @t)
+$  module    (unit @t)
+$  act       (unit @t)
+$  uship     (unit ship)
::
::  GroundSeg Activity
::
+$  activity      @t
::
::  Actions
::
+$  admin  $%  [%valve =valve]
               [%retry =retry]
               [%interval =interval]
               [%limit =limit]
           ==
+$  macro  $%  [%verify ~]
               [%login =password]
           ==
+$  raw    $%  [%send =action]
           ==
--
