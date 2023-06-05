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
    =reconnect-interval
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
+$  id                  @t
+$  pending             (map id created=@da)
+$  broadcast           @t
+$  action              @t
+$  activity            @t
+$  last-contact        @da
+$  reconnect-interval  @dr
+$  retry         ?
::
::  Actions
::
+$  agent    $%  [%action action]
                 [%connect retry reconnect-interval]
             ==
+$  earth    $%  [%broadcast broadcast]
                 [%activity activity]
             ==
+$  control  $%  [%unlocked ?]
                 [%interval reconnect-interval]
             ==
::
--
