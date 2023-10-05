# %groundseg
Gall agent that digests messages received from goseg via %lick and rebroadcasts it as a subscription

### Lick bridge (golang)
- Check for open lick ports for each pier in goseg
- Broadcast--bitstream? *still unclear how it should send the jammed nouns into lick*
- Receives actions that go through handler (same as ws but without the ws portion)
- Rejects certain action based on auth status (urbit/authorized)
- In pier settings, an extra option to login to groundseg

**Note**: `auth_status` in the broadcast structure will be by default "urbit" instead of "authorized". 
 This will only send the lick port what's inside broadcastState.Urbits[patp]. After the user has logged in however,
 proceed to provide the user with the full broadcastState.

**Another note**: This bypasses the token auth stuff. Should still wrap the actions in the same structure for simplicity though.

### groundseg.hoon
- Lick's `on-arvo` handles the message (should be a cord of json)
- Run it through our json to hoon mark. Store in state.
- Receives subs for specific paths (ie. `/profile/startram/info/registered`)
- Broadcast to said paths relevant information
- Receives actions as pokes
- Converts said pokes into a cord of json, spit it to the port via lick

### webui
- Basically gonna be the same as what we have in goseg, but with some additional urbit.js stuff that will be switch off based on the broadcast
