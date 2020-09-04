package datastructure

import "fmt"

//State 状态
type State struct {
	name   string
	trans  map[string]string //key是event value是state的name
	action func()
}

//Fsm 状态机
type Fsm struct {
	states  map[string]*State //key是state的name value是state指针
	current string            //currentEvent
}

//NewFsm 创建
func NewFsm(init string) *Fsm {
	m := new(Fsm)
	m.current = init
	m.states = make(map[string]*State)
	return m
}

//AddRule 添加规则
func (m *Fsm) AddRule(state string, event string, toState string, action func()) {
	if m.states[state] == nil {
		m.states[state] = new(State)
		m.states[state].name = state
		m.states[state].trans = make(map[string]string)
	}
	m.states[state].trans[event] = toState
	m.states[state].action = action
}

//OnEvent 事件发生的回调
func (m *Fsm) OnEvent(event string) {
	currentState := m.states[m.current]
	nextStateName, ok := currentState.trans[event]
	if !ok {
		fmt.Printf("stata:%s has no event %s\n", currentState.name, event)
		return
	}
	nextState := m.states[nextStateName]
	m.current = nextStateName
	nextState.action()
}
