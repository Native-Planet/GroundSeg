|%
::
+$  state-0  
  $:  %0
    session=?(%active %inactive)
    enabled=?
    =broadcast
    =pending
    =last
  ==
::
+$  earth  $%  [%broadcast broadcast]
               [%activity activity]
           ==
+$  mars  [%action action]
::
+$  id         @t
+$  pending    (map id created=@da)
+$  broadcast  @t
+$  action     @t
+$  activity   @t
+$  last       @da
::
--
