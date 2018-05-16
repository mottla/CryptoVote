package main

import "encoding/json"

type MessageType int

const (
	MessageTypeQueryLatest MessageType = iota
	MessageTypeQueryAll    MessageType = iota
	MessageTypeBlocks      MessageType = iota
	MessageTypeTransaction MessageType = iota
)

func (ms MessageType) name() string {
	switch ms {
	case MessageTypeQueryLatest:
		return "QUERY_LATEST"
	case MessageTypeQueryAll:
		return "QUERY_ALL"
	case MessageTypeBlocks:
		return "BLOCKS"
	case MessageTypeTransaction:
		return "TRANSACTIONS"
	default:
		return "UNKNOWN"
	}
}

type Message struct {
	Type       MessageType `json:"type"`
	Data       string      `json:"Data"`
	StartPoint uint32      `json:"start"`
}

func newQueryLatestMessage() *Message {
	return &Message{
		Type: MessageTypeQueryLatest,
	}
}

//var ctr = 0

func newQueryAllMessage(startPoint, buffer uint32) *Message {
	//ctr++
	//if ctr > 200 {
	//	panic("asdf")
	//}
	var sp uint32 = 1
	if startPoint <= 0 {
		sp = 1
	}
	if startPoint > 4294967294 {
		sp = 1
	}
	if startPoint > buffer {
		sp = uint32(startPoint) - buffer
	}
	return &Message{
		Type:       MessageTypeQueryAll,
		StartPoint: sp,
	}
}

func newBlocksMessage(blocks Blocks) (*Message, error) {
	b, err := json.Marshal(blocks)
	if err != nil {
		return nil, err
	}

	return &Message{
		Type: MessageTypeBlocks,
		Data: string(b),
	}, nil
}

func newTxMessage(transactions []transaction) (*Message, error) {
	b, err := json.Marshal(transactions)
	if err != nil {
		return nil, err
	}

	return &Message{
		Type: MessageTypeTransaction,
		Data: string(b),
	}, nil
}
