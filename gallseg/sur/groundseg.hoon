|%
::
::  State
::
+$  versioned-state
  $%
    state-0
  ==
::
+$  state-0  [%0 =settings =session =broadcast]
::
::  Types
::
+$  settings
  $:
    unlocked=?
    reconnect-interval=@ud
  ==
::
+$  session
  $:
    =last-contact
    =pending
    status=?(%active %inactive)
    token-id=@t
    token=@t
  ==
::
+$  id         @t
+$  pending    (map id created=@da)
+$  broadcast  @t
+$  action     @t
+$  activity   @t
+$  last       @da
::
::  Actions
::
+$  earth  $%  [%broadcast broadcast]
               [%activity activity]
           ==
+$  mars  [%action action]
::
--
