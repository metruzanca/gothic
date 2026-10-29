package main

import "github.com/metruzanca/gothic"

// Model is the application state.
type Model struct {
	Messages []string
}

// NewModel returns the initial model.
func NewModel() Model {
	return Model{Messages: []string{"ready"}}
}

// PingClicked is dispatched when the user clicks Ping.
type PingClicked struct{}

// PongReceived carries the procedure result.
type PongReceived struct{ Text string }

// PingFailed carries the procedure error.
type PingFailed struct{ Message string }

// Init starts no commands.
func (m Model) Init() gothic.Cmd {
	return nil
}

// Update handles messages and returns the next model and command.
func (m Model) Update(msg gothic.Msg) (gothic.Model, gothic.Cmd) {
	switch msg := msg.(type) {
	case PingClicked:
		m.Messages = append(m.Messages, "ping")
		return m, callPing
	case PongReceived:
		m.Messages = append(m.Messages, msg.Text)
	case PingFailed:
		m.Messages = append(m.Messages, msg.Message)
	}
	return m, nil
}

func callPing() gothic.Msg {
	pong, err := Ping()
	if err != nil {
		return PingFailed{Message: err.Error()}
	}
	return PongReceived{Text: pong}
}
