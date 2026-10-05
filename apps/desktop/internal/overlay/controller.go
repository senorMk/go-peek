// Package overlay coordinates shortcut toggles and timed hiding without desktop dependencies.
package overlay

import (
	"sync"
	"time"
)

// Controller owns app-initiated visibility. Native minimization is a separate window state.
// Show/hide callbacks must be nonblocking and must not call Controller methods.
type Controller struct {
	mu             sync.Mutex
	show, hide     func()
	timer          *time.Timer
	generation     uint64
	hidden, closed bool
	capturing      bool
}

func New(show, hide func()) *Controller { return &Controller{show: show, hide: hide} }

func (c *Controller) cancelRestore() {
	c.generation++
	if c.timer != nil {
		c.timer.Stop()
		c.timer = nil
	}
}

func (c *Controller) Toggle() {
	c.ToggleWithRestore(c.show)
}

// ToggleWithRestore uses a custom show action for this toggle only. Timed and
// capture restores still use the default show callback. The callback must be
// nonblocking, just like the callbacks passed to New.
func (c *Controller) ToggleWithRestore(show func()) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed || c.capturing {
		return
	}
	c.cancelRestore()
	if c.hidden {
		show()
	} else {
		c.hide()
	}
	c.hidden = !c.hidden
}

func (c *Controller) HideFor(duration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed || c.capturing || duration <= 0 {
		return
	}
	c.cancelRestore()
	c.hide()
	c.hidden = true
	generation := c.generation
	c.timer = time.AfterFunc(duration, func() {
		c.mu.Lock()
		defer c.mu.Unlock()
		if c.closed || generation != c.generation {
			return
		}
		c.show()
		c.hidden = false
		c.timer = nil
	})
}

// BeginCapture keeps the window hidden until the returned completion function runs.
// Visibility controls are suspended so they cannot reveal GoPeek in the screenshot.
func (c *Controller) BeginCapture() (func(), bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed || c.capturing {
		return nil, false
	}
	c.cancelRestore()
	c.capturing = true
	c.hidden = true
	c.hide()
	var once sync.Once
	return func() {
		once.Do(func() {
			c.mu.Lock()
			defer c.mu.Unlock()
			c.capturing = false
			if !c.closed {
				c.show()
				c.hidden = false
			}
		})
	}, true
}

func (c *Controller) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	c.cancelRestore()
}
