//go:build js

// The browser build (used for screenshots) has no updater.
package main

type UpdState struct {
	State, Latest, Notes, Err string
	Progress                  float64
	Managed                   bool
}

func (s UpdState) status() string { return "Version " + Version }

type Updater struct {
	st   UpdState
	Quit func()
}

func newUpdater() *Updater            { return &Updater{st: UpdState{State: "idle"}} }
func (u *Updater) snapshot() UpdState { return u.st }
func (u *Updater) Check(manual bool)  {}
func (u *Updater) Apply()             {}
func cleanupOldUpdate()               {}
