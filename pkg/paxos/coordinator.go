package paxos

// TODO temporary name
type coord struct {
	cx   *cluster
	outs Putter

	chanPutCloser
}

func newCoord(c *cluster, outs Putter) *coord {
	return &coord{
		cx:    c,
		outs:  outs,
		chanPutCloser: chanPutCloser(make(chan Msg)),
	}
}

func (c *coord) Close() {
	c.chanPutCloser.Close()
}

func (c *coord) process() {
	crnd := uint64(c.cx.SelfIndex())
	if crnd == 0 {
		crnd += uint64(c.cx.Len())
	}

	var target string
	var cval string
	var rsvps int
	var vr uint64
	var vv string

	// Wait for the very first proposal
	for in := range c.chanPutCloser {
		if in.Cmd() != propose {
			continue
		}
		target = proposeParts(in)
		c.outs.Put(newInvite(crnd))
		vr = 0
		vv = ""
		rsvps = 0
		cval = ""
		break
	}

	for in := range c.chanPutCloser {
		if closed(c.chanPutCloser) {
			goto Done
		}
		switch in.Cmd() {
		case rsvp:
			i, vrnd, vval := rsvpParts(in)

			if cval != "" {
				continue
			}

			if i < crnd {
				continue
			}

			if vrnd > vr {
				vr = vrnd
				vv = vval
			}

			rsvps++
			if rsvps >= c.cx.Quorum() {
				var v string

				if vr > 0 {
					v = vv
				} else {
					v = target
				}
				cval = v

				chosen := newNominate(crnd, v)
				c.outs.Put(chosen)
			}
		case propose:
			target = proposeParts(in)
			fallthrough
		case tick:
			crnd += uint64(c.cx.Len())
			c.outs.Put(newInvite(crnd))
			vr = 0
			vv = ""
			rsvps = 0
			cval = ""
		}
	}

Done:
	return
}
