/*
 *   Copyright 2020 Dmitry Kann
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package player

import (
	"fmt"
	"github.com/fhs/gompd/v2/mpd"
	"github.com/pkg/errors"
	"sync"
	"time"
)

// Connector encapsulates functionality for connecting to MPD and watch for its changes
type Connector struct {
	// MPD address
	mpdAddress string
	// MPD password
	mpdPassword string
	// Whether a connection is supposed to be kept alive
	stayConnected bool

	// MPD client instance
	mpdClient      *mpd.Client
	mpdClientMutex sync.Mutex

	// Last reported MPD status
	mpdStatus      mpd.Attrs
	mpdStatusMutex sync.Mutex

	// Connector's connect channel
	chConnectorConnect chan int
	// Connector's quit channel
	chConnectorQuit chan int

	// Watcher's start channel
	chWatcherStart chan int
	// Watcher's quit channel
	chWatcherQuit chan int
	// MPD watcher instance
	mpdWatcher *mpd.Watcher

	// Callback for connection status change notifications
	onStatusChange func()
	// Callback for periodic message notifications
	onHeartbeat func()
	// Callback for subsystem change notifications
	onSubsystemChange func(subsystem string)
}

func NewConnector(onStatusChange func(), onHeartbeat func(), onSubsystemChange func(subsystem string)) *Connector {
	return &Connector{
		mpdStatus:         mpd.Attrs{},
		onStatusChange:    onStatusChange,
		onHeartbeat:       onHeartbeat,
		onSubsystemChange: onSubsystemChange,
	}
}

// Start() initialises the connector
// stayConnected: whether the connection must be automatically re-established when lost
func (c *Connector) Start(mpdAddress, mpdPassword string, stayConnected bool) {
	c.mpdAddress = mpdAddress
	c.mpdPassword = mpdPassword
	c.stayConnected = stayConnected

	// Allocate signals
	c.chConnectorConnect = make(chan int)
	c.chConnectorQuit = make(chan int)
	c.chWatcherStart = make(chan int)
	c.chWatcherQuit = make(chan int)

	// Start the connect goroutine
	go c.connect()

	// Start the watch goroutine
	go c.watch()

	c.startConnecting()
}

// Status() returns the last known MPD status
func (c *Connector) Status() mpd.Attrs {
	c.mpdStatusMutex.Lock()
	defer c.mpdStatusMutex.Unlock()
	return c.mpdStatus
}

// Stop() signals the connector to shut down
func (c *Connector) Stop() {
	if c.chConnectorQuit != nil {
		close(c.chConnectorQuit)
		c.chConnectorQuit = nil
	}
	if c.chWatcherQuit != nil {
		close(c.chWatcherQuit)
		c.chWatcherQuit = nil
	}

	// Close the connection to MPD, if any
	c.IfConnected(func(client *mpd.Client) {
		log.Debug("Stop connector")
		errCheck(client.Close(), "Close() failed")
		c.mpdClient = nil
	})

	// Reset the status
	c.resetStatus()

	// Notify the callback
	c.onStatusChange()
}

// GetPlaylists() queries and returns a slice of playlist names available in MPD
func (c *Connector) GetPlaylists() []string {
	// Fetch the list of playlists
	var attrs []mpd.Attrs
	var err error
	c.IfConnected(func(client *mpd.Client) {
		attrs, err = client.ListPlaylists()
	})
	if errCheck(err, "ListPlaylists() failed") {
		return nil
	}

	// Convert attrs to a slice of strings
	names := make([]string, len(attrs))
	for i, a := range attrs {
		names[i] = a["playlist"]
	}
	return names
}

// IfConnected() runs MPD client code if there's a connection with MPD
func (c *Connector) IfConnected(funcIfConnected func(client *mpd.Client)) {
	c.mpdClientMutex.Lock()
	defer c.mpdClientMutex.Unlock()
	if c.mpdClient != nil {
		funcIfConnected(c.mpdClient)
	}
}

// IfConnectedElse() runs MPD client code if there's a connection with MPD and/or code if there's no connection
func (c *Connector) IfConnectedElse(funcIfConnected func(client *mpd.Client), funcIfDisconnected func()) {
	c.mpdClientMutex.Lock()
	defer c.mpdClientMutex.Unlock()
	switch {
	// Disconnected
	case c.mpdClient == nil && funcIfDisconnected != nil:
		funcIfDisconnected()
	// Connected
	case c.mpdClient != nil && funcIfConnected != nil:
		funcIfConnected(c.mpdClient)
	}
}

// IsConnected() returns whether there's a connection with MPD
func (c *Connector) IsConnected() bool {
	c.mpdClientMutex.Lock()
	defer c.mpdClientMutex.Unlock()
	return c.mpdClient != nil
}

// resetStatus() clears the current MPD status, thread-safely
func (c *Connector) resetStatus() {
	c.setStatus(&mpd.Attrs{})
}

// setStatus() sets the current MPD status, thread-safely
func (c *Connector) setStatus(attrs *mpd.Attrs) {
	c.mpdStatusMutex.Lock()
	defer c.mpdStatusMutex.Unlock()
	c.mpdStatus = *attrs
}

// startConnecting() signals the connector to initiate connection process
func (c *Connector) startConnecting() {
	go func() { c.chConnectorConnect <- 1 }()
}

// connect() takes care of establishing a connection to MPD
func (c *Connector) connect() {
	log.Debug("connect()")
	var heartbeatTicker = time.NewTicker(time.Second)
	for {
		select {
		// Request to connect
		case <-c.chConnectorConnect:
			log.Debug("Start connector")

			var wasntConnected, connected bool
			var status mpd.Attrs
			var err error
			// If disconnected
			c.IfConnectedElse(
				nil,
				func() {
					wasntConnected = true

					// Try to connect to MPD
					var client *mpd.Client
					if client, err = mpd.DialAuthenticated("tcp", c.mpdAddress, c.mpdPassword); err != nil {
						err = errors.Errorf("Dial() failed", err)
						return
					}

					// Actualise the status
					if status, err = client.Status(); err != nil {
						err = errors.Errorf("connect(): Status() failed", err)
						// Disconnect since we're not "fully connected"
						errCheck(client.Close(), "connect(): Close() failed")
						return
					}

					// Connection succeeded
					c.mpdClient = client
					connected = true

					// Start the watcher
					go func() { c.chWatcherStart <- 1 }()

				})

			// Update the status
			if wasntConnected {
				if err != nil {
					status = mpd.Attrs{"error": fmt.Sprint(err)}
				}
				c.setStatus(&status)

				// If connected, notify the callback
				if connected {
					c.onStatusChange()
				}
			}

		// Heartbeat tick
		case <-heartbeatTicker.C:
			c.IfConnectedElse(
				func(client *mpd.Client) {
					// Connection lost
					status, err := client.Status()
					if errCheck(err, "Status() failed: connection to MPD lost") {
						// Remove client connection
						c.mpdClient = nil

						// Clear the status
						c.resetStatus()

						// Re-attempt the connection if needed
						if c.stayConnected {
							c.startConnecting()
						}

					} else {
						// Connection is okay, store the status
						c.setStatus(&status)
					}
				},
				func() {
					// Clear the status
					c.resetStatus()

					// Re-attempt the connection if needed
					if c.stayConnected {
						c.startConnecting()
					}
				})

			// Notify the callback
			c.onHeartbeat()

		// Request to quit
		case <-c.chConnectorQuit:
			// Kill the heartbeat timer
			heartbeatTicker.Stop()
			return
		}
	}
}

// watch() watches MPD subsystem changes
func (c *Connector) watch() {
	log.Debug("watch()")
	var rewatchTimer *time.Timer
	var eventChannel chan string = nil
	var errorChannel chan error = nil
	for {
		select {
		// Request to watch
		case <-c.chWatcherStart:
			log.Debug("Start watcher")

			// Remove the timer
			rewatchTimer = nil

			// If no watcher yet
			if c.mpdWatcher == nil {
				watcher, err := mpd.NewWatcher("tcp", c.mpdAddress, c.mpdPassword)
				// Failed to connect
				if err != nil {
					log.Warning("Failed to watch MPD", err)
					// Schedule a reconnection
					rewatchTimer = time.AfterFunc(3*time.Second, func() {
						c.chWatcherStart <- 1
					})

					// Connection succeeded
				} else {
					c.mpdWatcher = watcher
					eventChannel = watcher.Event
					errorChannel = watcher.Error
				}
			}

		// Watcher's event
		case subsystem := <-eventChannel:
			// Provide an empty map as fallback
			status := mpd.Attrs{}

			// Request player status if there's a connection
			c.IfConnected(func(client *mpd.Client) {
				st, err := client.Status()
				if errCheck(err, "watch(): Status() failed") {
					return
				}
				status = st
			})

			// Update the MPD's status
			c.setStatus(&status)

			// Notify the callback
			c.onSubsystemChange(subsystem)

		// Watcher's error
		case err := <-errorChannel:
			log.Warning("Watcher error", err)

		// Request to quit
		case <-c.chWatcherQuit:
			// Kill the reconnection timer, if any
			if rewatchTimer != nil {
				rewatchTimer.Stop()
			}

			// Close the connection to MPD, if any
			if c.mpdWatcher != nil {
				log.Debug("Stop watcher")
				errCheck(c.mpdWatcher.Close(), "mpdWatcher.Close() failed")
				c.mpdWatcher = nil
			}
			return
		}
	}
}
