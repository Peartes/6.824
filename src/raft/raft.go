package raft

//
// this is an outline of the API that raft must expose to
// the service (or tester). see comments below for
// each of these functions for more details.
//
// rf = Make(...)
//   create a new Raft server.
// rf.Start(command interface{}) (index, term, isleader)
//   start agreement on a new log entry
// rf.GetState() (term, isLeader)
//   ask a Raft for its current term, and whether it thinks it is leader
// ApplyMsg
//   each time a new entry is committed to the log, each Raft peer
//   should send an ApplyMsg to the service (or tester)
//   in the same server.
//

import (
	//	"bytes"

	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	//	"6.824/labgob"
	"6.824/labrpc"
)

//
// as each Raft peer becomes aware that successive log entries are
// committed, the peer should send an ApplyMsg to the service (or
// tester) on the same server, via the applyCh passed to Make(). set
// CommandValid to true to indicate that the ApplyMsg contains a newly
// committed log entry.
//
// in part 2D you'll want to send other kinds of messages (e.g.,
// snapshots) on the applyCh, but set CommandValid to false for these
// other uses.
//
type ApplyMsg struct {
	CommandValid bool
	Command      interface{}
	CommandIndex int

	// For 2D:
	SnapshotValid bool
	Snapshot      []byte
	SnapshotTerm  int
	SnapshotIndex int
}

//
// A Go object implementing a single Raft peer.
//
type Raft struct {
	mu        sync.Mutex          // Lock to protect shared access to this peer's state
	peers     []*labrpc.ClientEnd // RPC end points of all peers
	persister *Persister          // Object to hold this peer's persisted state
	me        int                 // this peer's index into peers[]
	dead      int32               // set by Kill()

	// Your data here (2A, 2B, 2C).
	// Look at the paper's Figure 2 for a description of what
	// state a Raft server must maintain.
	currentTerm int64 // Peer current term
	votedFor int // Peer voted for during the current term
	state int // state of this peer: 0 -> Leader, 1 -> Follower or 2 -> Candidate
	applyChan chan ApplyMsg // channel for client to send apply msgs to
	logs []int // client logs for this peer 
	leaderLastTime time.Time // last time we heard from the leader
	sleepDuration time.Duration // How long before hosting an election?

	nextIndex map[int]int // For leaders to know the next log index to send to a server
	clientData bool // Use this to know when to send an append entry with real entries
}

// return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {
	// Your code here (2A).
	rf.mu.Lock()
	defer rf.mu.Unlock()
	DPrintf("GetState[%d] -> %d", rf.me, rf.state)
	return int(rf.currentTerm), rf.state == 0
}

//
// save Raft's persistent state to stable storage,
// where it can later be retrieved after a crash and restart.
// see paper's Figure 2 for a description of what should be persistent.
//
func (rf *Raft) persist() {
	// Your code here (2C).
	// Example:
	// w := new(bytes.Buffer)
	// e := labgob.NewEncoder(w)
	// e.Encode(rf.xxx)
	// e.Encode(rf.yyy)
	// data := w.Bytes()
	// rf.persister.SaveRaftState(data)
}


//
// restore previously persisted state.
//
func (rf *Raft) readPersist(data []byte) {
	if data == nil || len(data) < 1 { // bootstrap without any state?
		return
	}
	// Your code here (2C).
	// Example:
	// r := bytes.NewBuffer(data)
	// d := labgob.NewDecoder(r)
	// var xxx
	// var yyy
	// if d.Decode(&xxx) != nil ||
	//    d.Decode(&yyy) != nil {
	//   error...
	// } else {
	//   rf.xxx = xxx
	//   rf.yyy = yyy
	// }
}


//
// A service wants to switch to snapshot.  Only do so if Raft hasn't
// have more recent info since it communicate the snapshot on applyCh.
//
func (rf *Raft) CondInstallSnapshot(lastIncludedTerm int, lastIncludedIndex int, snapshot []byte) bool {

	// Your code here (2D).

	return true
}

// the service says it has created a snapshot that has
// all info up to and including index. this means the
// service no longer needs the log through (and including)
// that index. Raft should now trim its log as much as possible.
func (rf *Raft) Snapshot(index int, snapshot []byte) {
	// Your code here (2D).

}


//
// example RequestVote RPC arguments structure.
// field names must start with capital letters!
//
type RequestVoteArgs struct {
	// Your data here (2A, 2B).
	Term int // the current term of the candidate
	CandidateId int64 // the id of the candidate
	LastLogIndex int // the length of the candidate log entry
	LastLogTerm int // the term of the last log entry
}

//
// example RequestVote RPC reply structure.
// field names must start with capital letters!
//
type RequestVoteReply struct {
	// Your data here (2A).
	Term int // the current term of the voter
	VoteGranted bool // was the vote granted ?
}

//
// AppendEntry Request RPC struct
//
type AppendEntryArgs struct {
	Term int
	LeaderId int
	PrevLogIndex int
	PrevLogTerm int
	Entries []int 
	Heartbeat bool
}

//
// AppendEntryResponse struct
//
type AppendEntryResponse struct {
	Term int // The term of the server log entry was sent
	Success bool // Does the follower contain entry matching prevLogIndex and prevLogTerm ?
}

//
// example RequestVote RPC handler.
//
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	// Your code here (2A, 2B).
	rf.mu.Lock()
	defer rf.mu.Unlock()
	// DPrintf(rf.toString())

	if rf.killed() {
		reply.Term = int(rf.currentTerm)
		reply.VoteGranted = false
		return
	}

	if rf.currentTerm > int64(args.Term) {
		// this candidate is stale
		reply.Term = int(rf.currentTerm)
		reply.VoteGranted = false
		return
	}

	if rf.currentTerm == int64(args.Term) && rf.state == 2 {
		// I already voted for this term so rejecting this vote request
		// DPrintf("rejecting server %d vote request for term %d because I server %d already voted for %d", args.CandidateId, args.Term, rf.me, rf.votedFor)
		DPrintf("S[%d] X S[%d] T%d", rf.me, args.CandidateId, args.Term)
		reply.Term = int(rf.currentTerm)
		reply.VoteGranted = false
		return
	}

	if args.LastLogTerm > rf.logs[len(rf.logs) - 1] {
		// candidate has more recent log term so should be more recent
		rf.state = 1
		rf.currentTerm = int64(args.Term)
		rf.votedFor = int(args.CandidateId)
		rf.leaderLastTime = time.Now()
		// reply with a passing vote
		reply.Term = int(rf.currentTerm)
		reply.VoteGranted = true
		DPrintf("S[%d] -> S[%d] T%d", rf.me, args.CandidateId, args.Term)
		DPrintf(rf.toString())
		return
	} else if args.LastLogTerm == rf.logs[len(rf.logs) - 1] {
		if args.LastLogIndex >= len(rf.logs) {
			// candidate has more logs so is more recent than I
			rf.state = 1
			rf.currentTerm = int64(args.Term)
			rf.votedFor = int(args.CandidateId)
			rf.leaderLastTime = time.Now()
			// reply with a passing vote
			reply.Term = int(rf.currentTerm)
			reply.VoteGranted = true
			// DPrintf("vote request from server %d for term %d has a log index greater than ours %d so vote granted by server %d ", args.CandidateId, args.Term, args.LastLogTerm, rf.me)
			DPrintf("S[%d] -> S[%d] T%d", rf.me, args.CandidateId, args.Term)
			DPrintf(rf.toString())
			return
		}
	}
	reply.Term = int(rf.currentTerm)
	reply.VoteGranted = false
	DPrintf("vote request from server %d not granted by server %d", args.CandidateId, rf.me)
	DPrintf("args is %v\n \n and", args)
	DPrintf(rf.toString())
}

func (rf *Raft) AppendEntry(args *AppendEntryArgs, reply *AppendEntryResponse) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	DPrintf("Heartbeat me(%d) <- %d", rf.me, args.LeaderId)

	if rf.killed() {
		reply.Term = int(rf.currentTerm)
		reply.Success = false
		return
	}
	if args.Term < int(rf.currentTerm) {
		// Server is stale
		reply.Term = int(rf.currentTerm)
		reply.Success = false
		DPrintf("Stale Leader [%d]. I(%d) T[%d] X Beat", args.LeaderId, rf.me, rf.currentTerm)
		return
	}
	if len(rf.logs) - 1 < args.PrevLogIndex || rf.logs[args.PrevLogIndex] != args.PrevLogTerm {
		rf.leaderLastTime = time.Now()
		// Entry at previous log index has a different term to the leader, let's update our logs
		reply.Term = int(rf.currentTerm)
		reply.Success = false
		DPrintf("Our Log is not up to date. We need log from leader")
		return
	}
	reply.Success = true
	reply.Term = int(rf.currentTerm)
	copy(rf.logs[args.PrevLogIndex + 1:], args.Entries[:])
	rf.state = 1
	// Reset heartbeat time
	rf.leaderLastTime = time.Now()
	DPrintf("I(%d) set timeout", rf.me)
	// TODO: Implement commit update
}
//
// example code to send a RequestVote RPC to a server.
// server is the index of the target server in rf.peers[].
// expects RPC arguments in args.
// fills in *reply with RPC reply, so caller should
// pass &reply.
// the types of the args and reply passed to Call() must be
// the same as the types of the arguments declared in the
// handler function (including whether they are pointers).
//
// The labrpc package simulates a lossy network, in which servers
// may be unreachable, and in which requests and replies may be lost.
// Call() sends a request and waits for a reply. If a reply arrives
// within a timeout interval, Call() returns true; otherwise
// Call() returns false. Thus Call() may not return for a while.
// A false return can be caused by a dead server, a live server that
// can't be reached, a lost request, or a lost reply.
//
// Call() is guaranteed to return (perhaps after a delay) *except* if the
// handler function on the server side does not return.  Thus there
// is no need to implement your own timeouts around Call().
//
// look at the comments in ../labrpc/labrpc.go for more details.
//
// if you're having trouble getting RPC to work, check that you've
// capitalized all field names in structs passed over RPC, and
// that the caller passes the address of the reply struct with &, not
// the struct itself.
//
func (rf *Raft) sendRequestVote(server int, args *RequestVoteArgs, reply *RequestVoteReply) bool {
	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
	return ok
}

func (rf *Raft) sendAppendEntry(server int, args *AppendEntryArgs, reply *AppendEntryResponse) bool {
	ok := rf.peers[server].Call("Raft.AppendEntry", args, reply)
	return ok
}

// Send out periodic heartbeats to all followers if we are leader
func (rf *Raft) sendHeartBeat() {
	go func ()  {
		for !rf.killed() {
			rf.mu.Lock()
			if rf.state == 0 {
				DPrintf("I(%d) term %d lead send beat", rf.me, rf.currentTerm)
				heartBeat := !rf.clientData
				if rf.clientData {
					rf.logs = append(rf.logs, int(rf.currentTerm))
					rf.clientData = false
				}
				rf.mu.Unlock()
				for i := range rf.peers {
					// Send out heartbeat to each server
					if rf.me != i {
						go func (i int, heartBeat bool)  {
							rf.mu.Lock()
							var appendEntry *AppendEntryArgs
							if heartBeat {
								// TODO: Here we need to send logs as needed to each servers
								appendEntry = &AppendEntryArgs{
									int(rf.currentTerm),
									rf.me,
									rf.nextIndex[i] - 1,
									rf.logs[rf.nextIndex[i] - 1],
									rf.logs[rf.nextIndex[i]:],
									heartBeat,
								}
							} else {
								// This is just an heartbeat. We send just last entry the server should have
								appendEntry = &AppendEntryArgs{
									int(rf.currentTerm),
									rf.me,
									rf.nextIndex[i] - 2,
									rf.logs[rf.nextIndex[i] - 2],
									rf.logs[rf.nextIndex[i] - 1:],
									heartBeat,
								}
							}
							rf.mu.Unlock()
							appendEntryResponse := &AppendEntryResponse{}
							DPrintf("sending beat [%d] -> [%d]", rf.me, i)
							ok := rf.sendAppendEntry(i, appendEntry, appendEntryResponse)
							rf.mu.Lock()
							defer rf.mu.Unlock()
							if ok && appendEntryResponse.Term > int(rf.currentTerm) {
								// There must be a new leader. Step down
								rf.state = 1
							} else if ok && appendEntryResponse.Success && !heartBeat {
								rf.nextIndex[i]++
							} else if ok && !appendEntryResponse.Success {
								// Follower is most likely stale on log
								rf.nextIndex[i]--
							}
						}(i, heartBeat)
					}
				}
				// TODO: Check for committing the entry from major response
				time.Sleep(time.Millisecond * 100)
			} else {
				rf.mu.Unlock()
				break
			}
		}
	}()
}
//
// the service using Raft (e.g. a k/v server) wants to start
// agreement on the next command to be appended to Raft's log. if this
// server isn't the leader, returns false. otherwise start the
// agreement and return immediately. there is no guarantee that this
// command will ever be committed to the Raft log, since the leader
// may fail or lose an election. even if the Raft instance has been killed,
// this function should return gracefully.
//
// the first return value is the index that the command will appear at
// if it's ever committed. the second return value is the current
// term. the third return value is true if this server believes it is
// the leader.
//
func (rf *Raft) Start(command interface{}) (int, int, bool) {
	index := -1
	term := -1
	isLeader := true

	// Your code here (2B).


	return index, term, isLeader
}

//
// the tester doesn't halt goroutines created by Raft after each test,
// but it does call the Kill() method. your code can use killed() to
// check whether Kill() has been called. the use of atomic avoids the
// need for a lock.
//
// the issue is that long-running goroutines use memory and may chew
// up CPU time, perhaps causing later tests to fail and generating
// confusing debug output. any goroutine with a long-running loop
// should call killed() to check whether it should stop.
//
func (rf *Raft) Kill() {
	atomic.StoreInt32(&rf.dead, 1)
	// Your code here, if desired.
	rf.mu.Lock()
	defer rf.mu.Unlock()
	DPrintf("I(%d) have been killed", rf.me)
	rf.state = 1
}

func (rf *Raft) killed() bool {
	z := atomic.LoadInt32(&rf.dead)
	return z == 1
}

// The ticker go routine starts a new election if this peer hasn't received
// heartsbeats recently.
func (rf *Raft) ticker() {
	electionTimeoutCond := sync.NewCond(&rf.mu)
	go func ()  {
		// checks if the last time we heard from the leader
		// is greater than the sleep duration
		// if it is then we create and election
		// if not we reset the sleep Duration
		for {
			time.Sleep(time.Millisecond * 400) // check for leader ping every 300ms
			electionTimeoutCond.Broadcast()
		}
	}()
	for !rf.killed() {
		// Your code here to check if a leader election should
		// be started and to randomize sleeping time using
		// time.Sleep().
		
		// Once we are leader, we need not set an election timeout anymore
		rf.mu.Lock()
		if rf.state != 0 {
			rf.leaderLastTime = time.Now()
			rand.Seed(time.Now().UnixNano())
			sleepRandDuration := rand.Int31n(2000) + 500 // generate a random number from 1000 - 3000
			sleepDuration := time.Millisecond * time.Duration(sleepRandDuration)
			rf.sleepDuration = sleepDuration
			// Kick of an election after timing out
			for time.Since(rf.leaderLastTime) < sleepDuration {
				DPrintf("I(%d) leader heartbeat %d", rf.me, time.Since(rf.leaderLastTime))
				electionTimeoutCond.Wait() // release the lock
			}
			// We have not heard from the leader for a while, let's start an election
			rf.currentTerm += 1
			rf.state = 2
			rf.votedFor = rf.me
			DPrintf("I(%d) new election [%d]", rf.me, rf.currentTerm)
			numOfVote := 1 // number of votes
			receivedVote := 1 // total number of received vote
			// Reset election timer
			rf.leaderLastTime = time.Now()
			cond := sync.NewCond(&rf.mu)
			
			requestVote := &RequestVoteArgs{
				int(rf.currentTerm),
				int64(rf.me),
				len(rf.logs),
				rf.logs[len(rf.logs) - 1],
			}
			rf.mu.Unlock()
			for i := range rf.peers {
				if rf.me != i {
					// let's not send a vote request to ourself
					go func (i int) {
						rf.mu.Lock()
						if rf.state == 2 {
							rf.mu.Unlock()
							reply := &RequestVoteReply{}
							rf.sendRequestVote(i, requestVote, reply)
							rf.mu.Lock()
							defer rf.mu.Unlock()
							receivedVote++
							if rf.currentTerm < int64(reply.Term) {
								// another leader must have been elected
								// step down and become a follower
								// DPrintf("seems my(%d) vote request for term %d was rejected becuase of a larger term entry of %d", rf.me, rf.currentTerm, reply.Term)
								rf.currentTerm = int64(reply.Term)
								rf.state = 1
							} else if reply.VoteGranted {
								// DPrintf("I(%d) got sever %d vote for term %d", rf.me, i, rf.currentTerm)
								numOfVote++
							}
							cond.Broadcast()
						} else {
							rf.mu.Unlock()
						}
					}(i)
				}
			}
			rf.mu.Lock()
			for numOfVote < len(rf.peers) / 2 + 1 && receivedVote < len(rf.peers) / 2 + 1 || rf.state != 2 {
				// DPrintf("I(%d) havent received enough votes for term %d to make me leader. I have %d currently and have received %d so far", rf.me, rf.currentTerm, numOfVote, receivedVote)
				cond.Wait()
			}
			if rf.state == 2 && numOfVote >= len(rf.peers) / 2 + 1 {
				// we won this election
				rf.state = 0
				rf.mu.Unlock()
				for i := range rf.peers {
					rf.nextIndex[i] = len(rf.logs)
				}
				rf.sendHeartBeat()
				// DPrintf("we %d are leader for term %d ", rf.me, rf.currentTerm)
			} else {
				rf.mu.Unlock()
				// DPrintf("I(%d) lost term [%d]", rf.me, rf.currentTerm)
			}
		} else {
			rf.mu.Unlock()
		}
	}
}

func (rf *Raft) toString() string {
	const debug = true
	if debug {
		return fmt.Sprintf("Raft server %d stat is \n current term -> %d \n votedFor -> %d \n state -> %d \n logs -> %v", rf.me, rf.currentTerm, rf.votedFor, rf.state, rf.logs);
	}
	return ""
}
//
// the service or tester wants to create a Raft server. the ports
// of all the Raft servers (including this one) are in peers[]. this
// server's port is peers[me]. all the servers' peers[] arrays
// have the same order. persister is a place for this server to
// save its persistent state, and also initially holds the most
// recent saved state, if any. applyCh is a channel on which the
// tester or service expects Raft to send ApplyMsg messages.
// Make() must return quickly, so it should start goroutines
// for any long-running work.
//
func Make(peers []*labrpc.ClientEnd, me int,
	persister *Persister, applyCh chan ApplyMsg) *Raft {
	rf := &Raft{}
	rf.peers = peers
	rf.persister = persister
	rf.me = me

	// Your initialization code here (2A, 2B, 2C).
	rf.applyChan = applyCh
	rf.currentTerm = 0
	rf.votedFor = me
	rf.state = 1
	rf.logs = []int {0}
	rf.leaderLastTime = time.Now()
	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())
	rf.clientData = false
	rf.nextIndex = map[int]int{}

	// start ticker goroutine to start elections
	go rf.ticker()


	return rf
}
