package main

import (
	"flag"
	"net"
	"os"

	"junta/paxos"
	"junta/proc"
	"junta/store"
	"junta/util"
	"junta/client"
	"junta/server"
)

const (
	alpha = 50
	idBits = 160
)


// Flags
var (
	listenAddr *string = flag.String("l", "[::1]:8040", "The address to bind to.")
	attachAddr *string = flag.String("a", "", "The address to bind to.")
)

func activate(ch chan store.Event) {
	// TODO implement this
	close(ch)
}

func main() {
	flag.Parse()

	util.LogWriter = os.Stderr

	outs := make(paxos.ChanPutCloser)

	self := util.RandHexString(idBits)
	st := store.New()
	seqn := uint64(0)
	if *attachAddr != "" {
		c, err := client.Dial(*attachAddr)
		if err != nil {
			panic(err)
		}

		var snap string
		seqn, snap, err = client.Join(c, self, *listenAddr)
		if err != nil {
			panic(err)
		}

		ch := make(chan store.Event)
		st.Wait(seqn + alpha, ch)
		st.Apply(1, snap)
		go activate(ch)

		// TODO sink needs a way to pick up missing values if there are any
		// gaps in its sequence
	} else {
		seqn = addMember(st, seqn + 1, self, *listenAddr)
		seqn = claimSlot(st, seqn + 1, "1", self)
		seqn = claimLeader(st, seqn + 1, self)
	}
	mg := paxos.NewManager(self, seqn, alpha, st, outs)

	if *attachAddr == "" {
		// Skip ahead alpha steps so that the registrar can provide a
		// meaningful cluster.
		for i := seqn + 1; i < seqn + alpha; i++ {
			go st.Apply(i, "") // nop
		}
	}

	listener, err := net.Listen("tcp", *listenAddr)
	if err != nil {
		panic(err)
	}

	sv := &server.Server{*listenAddr, st, mg}

	go func() {
		panic(proc.Monitor(self, st))
	}()

	go func() {
		panic(sv.Serve(listener))
	}()

	go func() {
		panic(sv.ListenAndServeUdp(outs))
	}()

	for {
		st.Apply(mg.Recv())
	}
}

func addMember(st *store.Store, seqn uint64, self, addr string) uint64 {
	// TODO pull out path as a const
	mx, err := store.EncodeSet("/j/junta/members/"+self, addr, store.Missing)
	if err != nil {
		panic(err)
	}
	st.Apply(seqn, mx)
	return seqn
}

func claimSlot(st *store.Store, seqn uint64, slot, self string) uint64 {
	// TODO pull out path as a const
	mx, err := store.EncodeSet("/j/junta/slot/"+slot, self, store.Missing)
	if err != nil {
		panic(err)
	}
	st.Apply(seqn, mx)
	return seqn
}

func claimLeader(st *store.Store, seqn uint64, self string) uint64 {
	// TODO pull out path as a const
	mx, err := store.EncodeSet("/j/junta/leader", self, store.Missing)
	if err != nil {
		panic(err)
	}
	st.Apply(seqn, mx)
	return seqn
}
