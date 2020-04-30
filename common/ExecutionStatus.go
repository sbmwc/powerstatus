package common

import ()

type ExecutionStatus struct {
	ErrString       string
	WarnMsgs        []string
	MsgIdsProcessed []string
}

func (e *ExecutionStatus) addMsgId(msgId string) {
	e.MsgIdsProcessed = append(e.MsgIdsProcessed, msgId)
}

func (e *ExecutionStatus) addWarnMsg(msg string) {
	e.WarnMsgs = append(e.WarnMsgs, msg)
}
