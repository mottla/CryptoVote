// Copyright (c) 2018-2019 by mottla
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
package main

import (
	"fmt"
	"bytes"

	"crypto/ecdsa"

	"github.com/CryptoVote/CryptoVote/CryptoNote1/edwards"
	"github.com/CryptoVote/CryptoVote/CryptoNote1"
	"time"
	"golang.org/x/crypto/sha3"
)

var (
	cfg *config
)

// See loadConfig for details on the configuration load process.
//todo copied this from papa bitcoin^^ would be nice if the project would find all those config settings implemented.. long way...
type config struct {
	ShowVersion          bool          `short:"V" long:"version" description:"Display version information and exit"`
	ConfigFile           string        `short:"C" long:"configfile" description:"Path to configuration file"`
	DataDir              string        `short:"b" long:"datadir" description:"Directory to store data"`
	LogDir               string        `long:"logdir" description:"Directory to log output."`
	AddPeers             []string      `short:"a" long:"addpeer" description:"Add a peer to connect with at startup"`
	ConnectPeers         []string      `long:"connect" description:"Connect only to the specified peers at startup"`
	DisableListen        bool          `long:"nolisten" description:"Disable listening for incoming connections -- NOTE: Listening is automatically disabled if the --connect or --proxy options are used without also specifying listen interfaces via --listen"`
	Listeners            []string      `long:"listen" description:"Add an interface/port to listen for connections (default all interfaces port: 8333, testnet: 18333)"`
	MaxPeers             int           `long:"maxpeers" description:"Max number of inbound and outbound peers"`
	DisableBanning       bool          `long:"nobanning" description:"Disable banning of misbehaving peers"`
	BanDuration          time.Duration `long:"banduration" description:"How long to ban misbehaving peers.  Valid time units are {s, m, h}.  Minimum 1 second"`
	BanThreshold         uint32        `long:"banthreshold" description:"Maximum allowed ban score before disconnecting and banning misbehaving peers."`
	Whitelists           []string      `long:"whitelist" description:"Add an IP network or IP that will not be banned. (eg. 192.168.1.0/24 or ::1)"`
	RPCUser              string        `short:"u" long:"rpcuser" description:"Username for RPC connections"`
	RPCPass              string        `short:"P" long:"rpcpass" default-mask:"-" description:"Password for RPC connections"`
	RPCLimitUser         string        `long:"rpclimituser" description:"Username for limited RPC connections"`
	RPCLimitPass         string        `long:"rpclimitpass" default-mask:"-" description:"Password for limited RPC connections"`
	RPCListeners         []string      `long:"rpclisten" description:"Add an interface/port to listen for RPC connections (default port: 8334, testnet: 18334)"`
	RPCCert              string        `long:"rpccert" description:"File containing the certificate file"`
	RPCKey               string        `long:"rpckey" description:"File containing the certificate key"`
	RPCMaxClients        int           `long:"rpcmaxclients" description:"Max number of RPC clients for standard connections"`
	RPCMaxWebsockets     int           `long:"rpcmaxwebsockets" description:"Max number of RPC websocket connections"`
	RPCMaxConcurrentReqs int           `long:"rpcmaxconcurrentreqs" description:"Max number of concurrent RPC requests that may be processed concurrently"`
	RPCQuirks            bool          `long:"rpcquirks" description:"Mirror some JSON-RPC quirks of Bitcoin Core -- NOTE: Discouraged unless interoperability issues need to be worked around"`
	DisableRPC           bool          `long:"norpc" description:"Disable built-in RPC server -- NOTE: The RPC server is disabled by default if no rpcuser/rpcpass or rpclimituser/rpclimitpass is specified"`
	DisableTLS           bool          `long:"notls" description:"Disable TLS for the RPC server -- NOTE: This is only allowed if the RPC server is bound to localhost"`
	DisableDNSSeed       bool          `long:"nodnsseed" description:"Disable DNS seeding for peers"`
	ExternalIPs          []string      `long:"externalip" description:"Add an ip to the list of local addresses we claim to listen on to peers"`
	Proxy                string        `long:"proxy" description:"Connect via SOCKS5 proxy (eg. 127.0.0.1:9050)"`
	ProxyUser            string        `long:"proxyuser" description:"Username for proxy server"`
	ProxyPass            string        `long:"proxypass" default-mask:"-" description:"Password for proxy server"`
	OnionProxy           string        `long:"onion" description:"Connect to tor hidden services via SOCKS5 proxy (eg. 127.0.0.1:9050)"`
	OnionProxyUser       string        `long:"onionuser" description:"Username for onion proxy server"`
	OnionProxyPass       string        `long:"onionpass" default-mask:"-" description:"Password for onion proxy server"`
	NoOnion              bool          `long:"noonion" description:"Disable connecting to tor hidden services"`
	TorIsolation         bool          `long:"torisolation" description:"Enable Tor stream isolation by randomizing user credentials for each connection."`
	TestNet3             bool          `long:"testnet" description:"Use the test network"`
	RegressionTest       bool          `long:"regtest" description:"Use the regression test network"`
	SimNet               bool          `long:"simnet" description:"Use the simulation test network"`
	AddCheckpoints       []string      `long:"addcheckpoint" description:"Add a custom checkpoint.  Format: '<height>:<hash>'"`
	DisableCheckpoints   bool          `long:"nocheckpoints" description:"Disable built-in checkpoints.  Don't do this unless you know what you're doing."`
	DbType               string        `long:"dbtype" description:"Database backend to use for the Block Chain"`
	Profile              string        `long:"profile" description:"Enable HTTP profiling on given port -- NOTE port must be between 1024 and 65536"`
	CPUProfile           string        `long:"cpuprofile" description:"Write CPU profile to the specified file"`
	DebugLevel           string        `short:"d" long:"debuglevel" description:"Logging level for all subsystems {trace, debug, info, warn, error, critical} -- You may also specify <subsystem>=<level>,<subsystem2>=<level>,... to set the log level for individual subsystems -- Use show to list available subsystems"`
	Upnp                 bool          `long:"upnp" description:"Use UPnP to map our listening port outside of NAT"`
	MinRelayTxFee        float64       `long:"minrelaytxfee" description:"The minimum transaction fee in BTC/kB to be considered a non-zero fee."`
	FreeTxRelayLimit     float64       `long:"limitfreerelay" description:"Limit relay of transactions with no transaction fee to the given amount in thousands of bytes per minute"`
	NoRelayPriority      bool          `long:"norelaypriority" description:"Do not require free or low-fee transactions to have high priority for relaying"`
	MaxOrphanTxs         int           `long:"maxorphantx" description:"Max number of orphan transactions to keep in memory"`
	Generate             bool          `long:"generate" description:"Generate (mine) bitcoins using the CPU"`
	MiningAddrs          []string      `long:"miningaddr" description:"Add the specified payment address to the list of addresses to use for generated blocks -- At least one address is required if the generate option is set"`
	BlockMinSize         uint32        `long:"blockminsize" description:"Mininum block size in bytes to be used when creating a block"`
	BlockMaxSize         uint32        `long:"blockmaxsize" description:"Maximum block size in bytes to be used when creating a block"`
	BlockMinWeight       uint32        `long:"blockminweight" description:"Mininum block weight to be used when creating a block"`
	BlockMaxWeight       uint32        `long:"blockmaxweight" description:"Maximum block weight to be used when creating a block"`
	BlockPrioritySize    uint32        `long:"blockprioritysize" description:"Size in bytes for high-priority/low-fee transactions when creating a block"`
	UserAgentComments    []string      `long:"uacomment" description:"Comment to add to the user agent -- See BIP 14 for more information."`
	NoPeerBloomFilters   bool          `long:"nopeerbloomfilters" description:"Disable bloom filtering support"`
	SigCacheMaxSize      uint          `long:"sigcachemaxsize" description:"The maximum number of entries in the signature verification cache"`
	BlocksOnly           bool          `long:"blocksonly" description:"Do not accept transactions from remote peers."`
	TxIndex              bool          `long:"txindex" description:"Maintain a full hash-based transaction index which makes all transactions available via the getrawtransaction RPC"`
	DropTxIndex          bool          `long:"droptxindex" description:"Deletes the hash-based transaction index from the database on start up and then exits."`
	AddrIndex            bool          `long:"addrindex" description:"Maintain a full address-based transaction index which makes the searchrawtransactions RPC available"`
	DropAddrIndex        bool          `long:"dropaddrindex" description:"Deletes the address-based transaction index from the database on start up and then exits."`
	RelayNonStd          bool          `long:"relaynonstd" description:"Relay non-standard transactions regardless of the default settings for the active network."`
	RejectNonStd         bool          `long:"rejectnonstd" description:"Reject non-standard transactions regardless of the default settings for the active network."`
}

func errorCall(msg string) (err error) {
	return &errorString{msg}
}

var emptychain = newBlockchain()

func (n *Node) validateTransaction(trans *transaction, bufferChain *Blockchain) (err error) {

	if bufferChain == nil {
		bufferChain = emptychain
	}

	if n.blockchain.hasTransaction(trans) || bufferChain.hasTransaction(trans) {
		return errorCall("Transaction with this signature is already on chain")
	}

	switch trans.Typ {

	case ADD_VOTERS:
		if !trans.verifySignature(n.edcurve) {
			return errorCall("Invalid signature! Transaction rejected")
		}
		//a transactionByID to only to specify a set of unseperable voteSetMap

		for _, pub := range trans.PubKeys {
			_, err := edwards.ParsePubKey(n.edcurve, pub[:])
			if err != nil {
				return errorCall("Key parsing failed")
			}
		}
		return nil

	case CREATE_VOTING:
		if !trans.verifySignature(n.edcurve) {
			return errorCall("Invalid signature! Transaction rejected")
		}
		//transactionByID to create a voting
		if len(trans.VoteSet) == 0 {
			return errorCall("Votingset empty")
		}
		if len(trans.PubKeys) == 0 {
			return errorCall("Missing public address/es to vote on")
		}
		//check if all named votesets are onchain
		for i, _ := range trans.VoteSet {
			_, ok1 := n.blockchain.BlockHoldingTx(trans.VoteSet[i])
			_, ok2 := bufferChain.BlockHoldingTx(trans.VoteSet[i])
			if !ok1 && !ok2 {
				return errorCall("Votingset referenz hash is not on the blockchain")
			}
		}
		//check if all candidates are vote-abel
		var err error
		for i, _ := range trans.PubKeys {
			//parse and validateTransaction thereby
			_, err = edwards.ParsePubKey(n.edcurve, trans.PubKeys[i][:])
			if err != nil {
				return errorCall(fmt.Sprintf("voting destination %x invalid: %s", trans.PubKeys[i], err.Error()))
			}

		}

		return nil

	case REVEAL_VOTING:
		if !trans.verifySignature(n.edcurve) {
			return errorCall("Invalid signature! Transaction rejected")
		}

		chainHoldingVote := n.blockchain
		block, ok := chainHoldingVote.BlockHoldingTx(trans.VoteID)

		if !ok {
			chainHoldingVote = bufferChain
			block, ok = chainHoldingVote.BlockHoldingTx(trans.VoteID)
			if !ok {
				return errorCall("voting referenz hash is not on the blockchain")
			}
		}
		if block.Data.Typ != CREATE_VOTING {
			return errorCall("voting referenz hash is not a voting-Contract")
		}
		if len(block.Data.PubKeys) != len(trans.PrivateKeys) {
			return errorCall(fmt.Sprintf("voting Contract has %v, yet transaction has %v elements!", len(block.Data.VoteSet), len(trans.PrivateKeys)))
		}
		if !block.Data.RevealNeeded {
			return errorCall("voting-Contract needs no key revealing")
		}

		if chainHoldingVote.votingMap[trans.VoteID].IsRevealed() {
			return errorCall("voting already revealed")
		}

		//Note that the revealer has to consider the order in which he publishes the privatekeys
		for i, _ := range block.Data.PubKeys {
			//take the transaction VoteSet entry and check if the secret parses to the public key that
			//were reveald in the CreateVote
			_, PUBfromTx, err := edwards.PrivKeyFromScalar(n.edcurve, trans.PrivateKeys[i][:])
			if err != nil {
				return err
			}
			if bytes.Compare(block.Data.PubKeys[i][:], PUBfromTx.Serialize()) != 0 {
				return errorCall(fmt.Sprintf("Key Missmatch at position %v", i))
			}
		}

	case VOTE:

		if sha3.Sum512(trans.toMsg().Bytes()) != trans.EdSignature {
			return errorCall("Invalid Hashsum! Voting-Transaction rejected")
		}
		chainHoldingVote := n.blockchain
		voteContract, ok := chainHoldingVote.Voting(trans.VoteID)

		if !ok {
			chainHoldingVote = bufferChain
			voteContract, ok = chainHoldingVote.Voting(trans.VoteID)
			if !ok {
				return errorCall("voting for a contract, that is not on chain")
			}
		}

		if voteContract.IsRevealed() {
			//should consider timestamps and/or index as well.
			return errorCall("voting already revealed. You voted too late.")
		}

		//checking for double spend attempt
		//TODO pasing the two bigInt into one 32byte might lead to unwanted sideffects..
		keyImage := edwards.BigIntPointToEncodedBytes(trans.Signature.Ix, trans.Signature.Iy)

		if voteContract.IsDoubleSpend(*keyImage) {
			return errorCall("voting KeyImage already found")
		}

		if !voteContract.RevealeNeeded() {
			if trans.RevealElement != [32]byte{} {
				return errorCall("Vote does not need Reveal-Element")
			}

			//if no reaveal is needed, everyone can see where the vote goes to
			//but not where it comes from. So we can check if the vote goes to an existing candidate
			//and reject the transaction if the candidate does not exist.
			if !voteContract.AllowedCandidate(trans.VoteTo) {
				return errorCall("Voted on unknown address. voting rejected.")
			}
		} else {
			if trans.RevealElement == [32]byte{} {
				return errorCall("Vote needs Reveal-Element")
			}
		}

		//validating the ring signature procedure starts. This part is time-consuming and hence a system vulnerability
		var pubkeys = make([]*ecdsa.PublicKey, 0)

		keys, err := voteContract.AllAllowedVotersPubkeys()
		if err != nil {
			return err
		}
		//this is true, if the voter didnt specify any subsets he used for his ringsignature
		//we then take all allowed voteSetMap for this votingcontract for ringsig
		if len(trans.VoteSet) == 0 {

			for i := 0; i < len(keys); i++ {
				key, err := edwards.ParsePubKey(n.edcurve, keys[i][:])
				if err != nil {
					return errorCall("Key parsing failed")
				}
				pubkeys = append(pubkeys, key.ToECDSA())
			}
		} else {
			return errorCall("Subset selection for a LSAG signature not supported yet")
			//TODO if subset was selected
		}

		if len(pubkeys) != len(trans.Signature.Ri) {
			return errorCall(fmt.Sprintf("Missmatch in number of keys (%v) selected to verify the signature with %v cosigners ", len(pubkeys), len(trans.Signature.Ri)))
		}
		n.hasher.Reset()
		n.hasher.Write(trans.VoteID[:])
		n.hasher.Write(trans.VoteTo[:])
		message := n.hasher.Sum(nil)
		//n.log("Validating signature on %x \nSignature:%v", message, trans.Signature)
		sig := CryptoNote1.NewLSAG(nil, n.edcurve, n.hasher)

		sig.Sigma = trans.Signature
		if !sig.Verify(message, pubkeys) {
			return errorCall("Signature is invalid")
		}
		n.log("SIGNATURE IS VALID")
		return nil

	default:
		return errorCall("Transaction type no supported")
	}

	return
}

//asserts that the blocks index are in ascendingOrder
func (n *Node) validateChain(blocks Blocks) (msg *Message, broadcast bool, err error) {

	//sort.Sort(blocks)
	//sort.Reverse(blocks)
	//expect the blocks to be in ascending order. indexed  n,n+1,n+2..

	n.mu.Lock()
	defer n.mu.Unlock()
	latestStaleBlock := n.blockchain.getLatestBlock()

	//unecessary check..
	for i := 0; i < len(blocks)-1; i++ {
		if blocks[i].Index+1 != blocks[i+1].Index {
			panic("wrong order.")
		}
	}

	//trim away all known blocks
	//TODO seems like the origin of some weird behavoíour..
	//example in past: got 5 blocks. starting at 53. current height is 55
	//			then all blocks from 53 to 56 were removed due to the following map lookup..#
	//			the chain is now at height 57 and cannot be attached. The querry all now starts a ping race..
	//c := 0
	for i, _ := range blocks {
		if !n.blockchain.hasBlock(blocks[i].Hash) {
			//if blocks[i].Index > latestStaleBlock.Index {
			//	panic("this is impossible")
			//}
			//c++
			blocks = blocks[i:]
		}
	}

	//we knew each block already, lets do nothing
	if len(blocks) == 0 {
		return nil, false, errorCall("received chain already integrated")
	}

	chainHeight := latestStaleBlock.Index
	n.log("validate chain from index ", blocks[0].Index, " to ", blocks[len(blocks)-1].Index, ". Current height is ", chainHeight)

	if blocks[0].Index < 1 {
		return nil, false, errorCall("received chain start-index to low")
	}

	//we received a chain to high to attach to local bc
	if blocks[0].Index > chainHeight+1 {
		//TODO a malicious fullnode could force me to ping him back by sending trash. somtimes responding with newQueryAll results in a ping race..
		fmt.Println("a")
		return newQueryAllMessage(chainHeight, uint32(2+len(blocks)*20/100)), true, nil
	}

	//the leading block in the chain is below our latest stale block
	if blocks[len(blocks)-1].Index < chainHeight {
		// if the recived chain matches onto our chain, we asume that the senders state is outdated and we could send him our chain?
		//NOTE: obviously this can be exploited.. should I really send blocks after receiving 'useless' data.. better don't
		return nil, false, errorCall("received chain already outdated")
	}

	//if the received chain is not linkable to our chain, we ask the sender to send again, starting 3 blocks earlier
	//Note: this algo is very inefficient. imagine a node that is 1k blocks behind. More then 300 send-receive would be needed.
	//			even worse so far is, that we send the entire blocks, instead of just headers
	//		todo find an efficient way for nodes, to synch
	if err := blocks[0].isValidAncestor(n.blockchain.Block_ind(blocks[0].Index - 1)); err != nil {
		n.log("a invalid ancestor at height", blocks[0].Index)
		n.log(blocks[0])
		n.log(n.blockchain.Block_ind(blocks[0].Index - 1))
		n.logError(err)
		//return nil, false, errorCall("recived shit")
		return newQueryAllMessage(blocks[0].Index, uint32(2+len(blocks)*20/100)), true, nil
		//return nil, false, errorCall("recived chain cannot be attached to local ledger due to inconsistency with leading chain element")
	}

	//add all valid transactions to this chain. This is very importand
	//and allowes us, to verify transaction, which require transcations to be 'on chain' already, to be validatable  (e.g. cannot reveal a voting, if the voting contract does not exist yet..)
	tempChain := newBlockchain()

	//n.validateTransaction checks if transaction isEmpty. we check it outside
	if !blocks[0].Data.isEmpty() {
		if err := n.validateTransaction(&blocks[0].Data, tempChain); err != nil {
			n.log("invalid transactions in leading chain block ", err.Error())
			return nil, false, errorCall("recived chain cannot be attached to local ledger due to invalid transaction within")
		} else {
			tempChain.addBlockCareless(blocks[0])
		}
	}
	//we recived an alternative leading block. I should check how the pros handle that..
	if len(blocks) == 1 && blocks[0].Index == latestStaleBlock.Index {
		//if our block is older, we replace it
		//if our block is younger, we send it to the
		if latestStaleBlock.Timestamp < blocks[0].Timestamp {
			//our block was first
			msg, err := newBlocksMessage(Blocks{latestStaleBlock})
			if err != nil {
				return msg, false, nil
			}
		}

	}
	//we validateTransaction the partial chain and shorten the array if it is inconsistent within
	//note that an attacker could send long chains which then cant be added to the chain due to invalidity
	for i := 1; i < len(blocks); i++ {
		if err := blocks[i].isValidAncestor(blocks[i-1]); err != nil {
			n.log("invalid ancestor at height", blocks[i].Index)
			n.log(blocks[i-1])
			n.log(blocks[i])
			n.logError(err)
			//from here on, we cut the chain
			blocks = blocks[:i]
			break
		}
		if !blocks[i].Data.isEmpty() {
			if err := n.validateTransaction(&blocks[i].Data, tempChain); err != nil {
				n.log("invalid transactions in chain at block ", i, " with id ", blocks[i].Index, ". Msg:", err.Error())
				//cut the rest
				blocks = blocks[:i]
				break
			} else {
				tempChain.addBlockCareless(blocks[i])

			}
		}

	}

	//should not be possible to validateTransaction as true
	if len(blocks) == 0 {
		panic("len 0 should not happen")
		return nil, false, nil
	}

	for i, _ := range blocks {
		n.blockchain.addBlock(blocks[i], n.pool)
		if !blocks[i].Data.isEmpty() {
			n.log("included ", blocks[i].Data.Typ.name(), " at height ", blocks[i].Index)
		}

	}
	//n.log("included ", len(blocks), " from ", blocks[0].Index, " to ", blocks[len(blocks)-1].Index)
	if n.miner.IsMining() {
		n.miner.updateHight <- n.blockchain.chainHeight()
	}

	//now we broadcast all blocks, we truly included
	msg, err = newBlocksMessage(blocks[len(blocks)-1:])

	return msg, true, err
}
