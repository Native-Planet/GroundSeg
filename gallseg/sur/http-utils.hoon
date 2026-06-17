::  http-utils: HTTP/SSE encoding, error responses, and request utilities
::
/+  server, multipart
|%
::  General utilities
::
++  numb
  |=  a=@u
  ^-  tape
  ?:  =(0 a)  "0"
  %-  flop
  |-  ^-  tape
  ?:(=(0 a) ~ [(add '0' (mod a 10)) $(a (div a 10))])
::
++  hexn
  |=  a=@u
  ^-  tape
  ?:  =(0 a)  "0"
  %-  flop
  |-  ^-  tape
  ?:  =(0 a)
    ~
  =+  m=(mod a 16)
  :_  $(a (div a 16))
  ?:  (lth m 10)
    (add '0' m)
  (add 'a' (sub m 10))
::
::  Types
::
+$  sse-key  [id=(unit @t) event=(unit @t)]
::
+$  sse-event
  $:  id=(unit @t)
      event=(unit @t)
      data=wain
  ==
::
+$  sse-manx
  $:  id=(unit @t)
      event=(unit @t)
      =manx
  ==
::
+$  sse-json
  $:  id=(unit @t)
      event=(unit @t)
      =json
  ==
::
+$  sse-connection
  $:  started=@da
      site=(list @ta)
      args=(list [key=@t value=@t])
  ==
::
+$  byte-range
  $%  [%from-to start=@ud end=@ud]  :: bytes=0-1023
      [%from start=@ud]              :: bytes=1024-
      [%suffix length=@ud]           :: bytes=-500
  ==
::
+$  parts  (list [@t part:multipart])
::
::  Request detection
::
++  is-sse-request
  |=  req=inbound-request:eyre
  ^-  ?
  ?&  ?=(%'GET' method.request.req)
      .=  [~ 'text/event-stream']
      (get-header:http 'accept' header-list.request.req)
  ==
::
++  sse-last-id
  |=  req=inbound-request:eyre
  ^-  (unit @t)
  (get-header:http 'last-event-id' header-list.request.req)
::
::  URL parsing
::
::  +parse-url: parse URL into path segments and query args
::  Unlike parse-request-line, does NOT strip extensions from filenames.
::
++  parse-url
  |=  url=@t
  ^-  [site=path args=quay:eyre]
  %+  fall
    %+  rush  url
    ;~  plug
      ;~(pfix fas (more fas smeg:de-purl:html))
      yque:de-purl:html
    ==
  [~ ~]
::
::  URL encoding
::
++  encode-request-line
  |=  lin=request-line:server
  ^-  @t
  =/  path=tape
    %-  zing
    %+  turn  site.lin
    |=(seg=@t (weld "/" (en-urlt:html (trip seg))))
  =/  url=@t  (crip path)
  =?  url  ?=(^ ext.lin)  (cat 3 url (cat 3 '.' u.ext.lin))
  ?~  args.lin  url
  =/  query=tape
    %+  roll  `(list [@t @t])`args.lin
    |=  [[key=@t val=@t] acc=tape]
    =/  sep=tape  ?~(acc "?" "&")
    %+  weld  acc
    %+  weld  sep
    %+  weld  (en-urlt:html (trip key))
    %+  weld  "="
    (en-urlt:html (trip val))
  (cat 3 url (crip query))
::
::  SSE encoding
::
++  sse-header
  ^-  response-header:http
  :-  200
  :~  ['content-type' 'text/event-stream']
      ['cache-control' 'no-cache']
      ['connection' 'keep-alive']
  ==
::
++  sse-keep-alive  `octs`(as-octs:mimes:html ':\0a\0a')
::
++  sse-encode
  =|  comments=wain
  =|  retry=(unit @ud)
  |=  events=(list sse-event)
  ^-  octs
  =|  response=wain
  =?  response  ?=(^ retry)
    (snoc response (cat 3 'retry: ' (crip (numb u.retry))))
  =.  response
    |-
    ?~  events
      (snoc response '')
    =?  response  ?=(^ id.i.events)
      (snoc response (cat 3 'id: ' u.id.i.events))
    =?  response  ?=(^ event.i.events)
      (snoc response (cat 3 'event: ' u.event.i.events))
    =.  response
      %+  weld  response
      ?~  data.i.events
        ~['data: ']
      %+  turn  data.i.events
      |=(=@t (cat 3 'data: ' t))
    $(events t.events)
  =.  response
    |-
    ?~  comments
      (snoc response '')
    =.  response  (snoc response (cat 3 ': ' i.comments))
    $(comments t.comments)
  (as-octs:mimes:html (of-wain:format response))
::
::  Content conversion
::
++  manx-to-wain
  |=  =manx
  ^-  wain
  (to-wain:format (crip (en-xml:html manx)))
::
++  json-to-wain
  |=  =json
  ^-  wain
  [(en:json:html json)]~
::
::  Error rendering
::
++  render-tang-to-wall
  |=  [wid=@u tan=tang]
  ^-  wall
  (zing (turn tan |=(a=tank (wash 0^wid a))))
::
++  render-tang-to-marl
  |=  [wid=@u tan=tang]
  ^-  marl
  =/  raw=(list tape)  (zing (turn tan |=(a=tank (wash 0^wid a))))
  ::
  |-  ^-  marl
  ?~  raw  ~
  [;/(i.raw) ;br; $(raw t.raw)]
::
::  Response building
::
++  mime-response
  |=  =mime
  ^-  simple-payload:http
  :_  `q.mime
  :-  200
  :~  ['cache-control' 'no-cache']
      ['content-type' (rsh [3 1] (spat p.mime))]
  ==
::
++  two-oh-four
  ^-  simple-payload:http
  [[204 ['content-type' 'application/json']~] ~]
::
++  login-redirect
  |=  lin=request-line:server
  ^-  simple-payload:http
  =-  [[307 ['location' -]~] ~]
  %^  cat  3
    '/~/login?redirect='
  (encode-request-line lin)
::
++  internal-server-error
  |=  [authorized=? msg=tape t=tang]
  ^-  simple-payload:http
  =;  =manx
    :_  `(manx-to-octs:server manx)
    [500 ['content-type' 'text/html']~]
  ;html
    ;head
      ;title:"500 Internal Server Error"
    ==
    ;body
      ;h1:"Internal Server Error"
      ;p: {msg}
      ;*  ?:  authorized
            ;=
              ;code:"*{(render-tang-to-marl 80 t)}"
            ==
          ~
    ==
  ==
::
++  method-not-allowed
  |=  method=@t
  ^-  simple-payload:http
  =;  =manx
    :_  `(manx-to-octs:server manx)
    [405 ['content-type' 'text/html']~]
  ;html
    ;head
      ;title:"405 Method Not Allowed"
    ==
    ;body
      ;h1:"Method Not Allowed: {(trip method)}"
    ==
  ==
::
++  unsupported-browser
  ^-  simple-payload:http
  =;  =manx
    :_  `(manx-to-octs:server manx)
    [426 ['content-type' 'text/html']~]
  ;html
    ;head
      ;title:"426 Upgrade Required"
    ==
    ;body
      ;h1:"Modern Browser Required"
      ;p:"This application uses subdomain isolation for security."
      ;p:"Your browser must support Fetch Metadata (sec-fetch-mode and sec-fetch-site headers)."
      ;p:"Supported browsers:"
      ;ul
        ;li:"Brave 1.8+ (August 2019)"
        ;li:"Chrome/Edge 76+ (August 2019)"
        ;li:"Firefox 90+ (July 2021)"
        ;li:"Tor Browser 10.5+ (June 2021)"
        ;li:"Safari 16.4+ (March 2023)"
      ==
      ;p:"Please upgrade your browser to continue."
    ==
  ==
::
++  cross-origin-forbidden
  |=  [mode=(unit @t) site=(unit @t)]
  ^-  simple-payload:http
  =;  =manx
    :_  `(manx-to-octs:server manx)
    [403 ['content-type' 'text/html']~]
  ;html
    ;head
      ;title:"403 Forbidden"
    ==
    ;body
      ;h1:"Cross-Origin Request Blocked"
      ;p:"This application does not accept requests from other origins."
      ;*  ?~  mode  ~
          :_  ~
          ;p
            ; Sec-Fetch-Mode:
            ;code:"{(trip u.mode)}"
          ==
      ;*  ?~  site  ~
          :_  ~
          ;p
            ; Sec-Fetch-Site:
            ;code:"{(trip u.site)}"
          ==
    ==
  ==
::
::  Range request support
::
++  parse-range-header
  |=  headers=header-list:http
  ^-  (unit byte-range)
  =/  range-value=(unit @t)  (get-header:http 'range' headers)
  ?~  range-value
    ~
  ::  Strip "bytes=" prefix
  =/  val=tape  (trip u.range-value)
  ?.  =((scag 6 val) "bytes=")
    ~
  =/  range-part=tape  (slag 6 val)
  ::  Find the dash
  =/  dash-pos=(unit @ud)
    |-  ^-  (unit @ud)
    =+  pos=0
    |-  ^-  (unit @ud)
    ?~  range-part  ~
    ?:  =(i.range-part '-')  `pos
    $(range-part t.range-part, pos +(pos))
  ?~  dash-pos
    ~
  ::  Split on dash
  =/  before=tape  (scag u.dash-pos range-part)
  =/  after=tape  (slag +(u.dash-pos) range-part)
  ::  Parse three cases
  ?:  =(before ~)
    ::  bytes=-500 (suffix)
    =/  len=(unit @ud)  (rush (crip after) dem)
    ?~  len
      ~
    [~ %suffix u.len]
  ?:  =(after ~)
    ::  bytes=1024- (from)
    =/  start=(unit @ud)  (rush (crip before) dem)
    ?~  start
      ~
    [~ %from u.start]
  ::  bytes=0-1023 (from-to)
  =/  start=(unit @ud)  (rush (crip before) dem)
  =/  end=(unit @ud)  (rush (crip after) dem)
  ?.  &(?=(^ start) ?=(^ end))
    ~
  [~ %from-to u.start u.end]
::
++  slice-mime
  |=  [range=byte-range =mime]
  ^-  [content-range=@t data=octs]
  =/  total-size=@ud  p.q.mime
  ::  Calculate actual start/end positions
  =/  [rng-start=@ud rng-end=@ud]
    ?-  -.range
        %from-to
      [start.range end.range]
      ::
        %from
      [start.range (dec total-size)]
      ::
        %suffix
      =/  suf-start=@ud
        ?:  (gte length.range total-size)  0
        (sub total-size length.range)
      [suf-start (dec total-size)]
    ==
  ::  Ensure end doesn't exceed file size
  =/  actual-end=@ud  (min rng-end (dec total-size))
  ::  Calculate slice length
  =/  slice-len=@ud  +((sub actual-end rng-start))
  ::  Extract bytes and wrap with as-octs
  =/  =octs  (as-octs:mimes:html (cut 3 [rng-start slice-len] q.q.mime))
  ::  Build Content-Range header using rap
  =/  content-range=@t
    %+  rap  3
    :~  'bytes '
        (crip (a-co:co rng-start))  '-'
        (crip (a-co:co actual-end))  '/'
        (crip (a-co:co total-size))
    ==
  [content-range octs]
::
++  range-response
  |=  [=header-list:http =mime]
  ^-  simple-payload:http
  =/  range=(unit byte-range)  (parse-range-header header-list)
  ?~  range
    ::  No Range header - return full file with 200
    :_  `q.mime
    :-  200
    :~  ['cache-control' 'no-cache']
        ['accept-ranges' 'bytes']
        ['content-type' (rsh [3 1] (spat p.mime))]
    ==
  ::  Range header present - always return 206 with Content-Range
  =/  [content-range=@t data=octs]  (slice-mime u.range mime)
  :_  `data
  :-  206
  :~  ['content-range' content-range]
      ['content-length' (crip (a-co:co p.data))]
      ['cache-control' 'no-cache']
      ['accept-ranges' 'bytes']
      ['content-type' (rsh [3 1] (spat p.mime))]
  ==
::
::  Multipart form data
::
++  grab-part
  =|  lead=(list [@t part:multipart])
  |=  [key=@t parts=(list [@t part:multipart])]
  ^-  [(unit part:multipart) (list [@t part:multipart])]
  ?~  parts
    [~ (flop lead)]
  ?:  =(key -.i.parts)
    [[~ +.i.parts] (weld (flop lead) t.parts)]
  $(parts t.parts, lead [i.parts lead])
::
::  Gall card builders for HTTP/SSE
::
++  give-response-header
  |=  [eyre-id=@ta =response-header:http]
  ^-  card:agent:gall
  :^  %give  %fact  ~[/http-response/[eyre-id]]
  http-response-header+!>(response-header)
::
++  give-response-data
  |=  [eyre-id=@ta data=(unit octs)]
  ^-  card:agent:gall
  [%give %fact ~[/http-response/[eyre-id]] http-response-data+!>(data)]
::
++  kick-eyre-sub
  |=  eyre-id=@ta
  ^-  card:agent:gall
  [%give %kick ~[/http-response/[eyre-id]] ~]
::
++  give-sse-event
  |=  [eyre-id=@ta =sse-event]
  ^-  card:agent:gall
  =/  data=octs  (sse-encode ~[sse-event])
  (give-response-data eyre-id `data)
::
++  give-sse-manx
  |=  [eyre-id=@ta id=(unit @t) event=(unit @t) =manx]
  ^-  card:agent:gall
  =/  =sse-event  [id event (manx-to-wain manx)]
  (give-sse-event eyre-id sse-event)
::
++  give-sse-json
  |=  [eyre-id=@ta id=(unit @t) event=(unit @t) =json]
  ^-  card:agent:gall
  =/  =sse-event  [id event (json-to-wain json)]
  (give-sse-event eyre-id sse-event)
::
++  give-sse-header
  |=  eyre-id=@ta
  ^-  card:agent:gall
  (give-response-header eyre-id sse-header)
::
++  give-sse-keep-alive
  |=  eyre-id=@ta
  ^-  card:agent:gall
  (give-response-data eyre-id `sse-keep-alive)
::
++  give-manx-response
  |=  [eyre-id=@ta =manx]
  ^-  (list card:agent:gall)
  %+  give-simple-payload:app:server
    eyre-id
  (manx-response:gen:server manx)
::
++  give-mime-response
  |=  [eyre-id=@ta =mime]
  ^-  (list card:agent:gall)
  %+  give-simple-payload:app:server
    eyre-id
  (mime-response mime)
--
