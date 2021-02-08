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
	"github.com/fhs/gompd/v2/mpd"
	"github.com/pkg/errors"
	"sync"
	"time"
)

// Connector encapsulates functionality for connecting to MPD and watch for its changes
type Connector struct {
	mpdNetwork    string // MPD network
	mpdAddress    string // MPD address
	mpdPassword   string // MPD password
	stayConnected bool   // Whether a connection is supposed to be kept alive

	mpdClient           *mpd.Client // MPD client instance
	mpdClientConnecting bool        // Whether MPD connection is being established
	mpdClientMutex      sync.RWMutex

	mpdStatus      mpd.Attrs // Last reported MPD status
	mpdStatusMutex sync.RWMutex

	chConnectorConnect chan bool // Connector's connect channel
	chConnectorQuit    chan bool // Connector's quit channel

	chWatcherStart chan bool // Watcher's start channel
	chWatcherStop  chan bool // Watcher's suspend/quit channel

	onStatusChange    func()                 // Callback for connection status change notifications
	onHeartbeat       func()                 // Callback for periodic message notifications
	onSubsystemChange func(subsystem string) // Callback for subsystem change notifications
}

// NewConnector creates and returns a new Connector instance
func NewConnector(onStatusChange func(), onHeartbeat func(), onSubsystemChange func(subsystem string)) *Connector {
	return &Connector{
		mpdStatus:          mpd.Attrs{},
		onStatusChange:     onStatusChange,
		onHeartbeat:        onHeartbeat,
		onSubsystemChange:  onSubsystemChange,
		chConnectorConnect: make(chan bool),
		chConnectorQuit:    make(chan bool),
		chWatcherStart:     make(chan bool),
		chWatcherStop:      make(chan bool),
	}
}

// Start initialises the connector
// stayConnected: whether the connection must be automatically re-established when lost
func (c *Connector) Start(mpdNetwork, mpdAddress, mpdPassword string, stayConnected bool) {
	c.mpdNetwork = mpdNetwork
	c.mpdAddress = mpdAddress
	c.mpdPassword = mpdPassword
	c.stayConnected = stayConnected

	// Start the connect goroutine
	go c.connect()

	// Start the watch goroutine
	go c.watch()

	c.startConnecting()
}

// Status returns the last known MPD status
func (c *Connector) Status() mpd.Attrs {
	c.mpdStatusMutex.RLock()
	defer c.mpdStatusMutex.RUnlock()
	return c.mpdStatus
}

// Stop signals the connector to shut down
func (c *Connector) Stop() {
	// Ignore if not connected/connecting
	if connected, connecting := c.ConnectStatus(); !connected && !connecting {
		return
	}

	// Quit connector and watcher
	c.stayConnected = false
	c.chConnectorQuit <- true
	c.chWatcherStop <- true

	// Close the connection to MPD, if any
	c.mpdClientMutex.Lock()
	c.mpdClientConnecting = false
	if c.mpdClient != nil {
		log.Debug("Disconnect from MPD")
		errCheck(c.mpdClient.Close(), "Close() failed")
		c.mpdClient = nil
	}
	c.mpdClientMutex.Unlock()

	// Reset the status
	c.setStatus(mpd.Attrs{})

	// Notify the callback
	c.onStatusChange()
}

// GetPlaylists queries and returns a slice of playlist names available in MPD
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

// IfConnected runs MPD client code if there's a connection with MPD
func (c *Connector) IfConnected(funcIfConnected func(client *mpd.Client)) {
	c.mpdClientMutex.RLock()
	defer c.mpdClientMutex.RUnlock()
	if c.mpdClient != nil {
		funcIfConnected(c.mpdClient)
	}
}

// IsConnected returns whether there's a connection with MPD and whether it's being established
func (c *Connector) ConnectStatus() (bool, bool) {
	c.mpdClientMutex.RLock()
	defer c.mpdClientMutex.RUnlock()
	return c.mpdClient != nil, c.mpdClientConnecting
}

// setStatus sets the current MPD status, thread-safely
func (c *Connector) setStatus(attrs mpd.Attrs) {
	c.mpdStatusMutex.Lock()
	defer c.mpdStatusMutex.Unlock()
	c.mpdStatus = attrs
}

// startConnecting signals the connector to initiate connection process
func (c *Connector) startConnecting() {
	go func() { c.chConnectorConnect <- true }()
}

// connect maintains MPD connection and invokes callbacks until something is sent via chConnectorQuit
func (c *Connector) connect() {
	log.Debug("connect()")
	var heartbeatTicker = time.NewTicker(time.Second)
	for {
		select {
		// Request to connect
		case <-c.chConnectorConnect:
			c.doConnect(true, false)

		// Heartbeat tick
		case <-heartbeatTicker.C:
			c.doConnect(false, true)

		// Request to quit
		case <-c.chConnectorQuit:
			// Kill the heartbeat timer
			heartbeatTicker.Stop()
			return
		}
	}
}

// doConnect takes care of (re)establishing a connection to MPD and calling the status/heartbeat callbacks
func (c *Connector) doConnect(connect, heartbeat bool) {
	var err error
	var client *mpd.Client
	var wasConnected bool
	connected, _ := c.ConnectStatus()

	// If there's a request to connect and not connected yet
	if connect && !connected {
		// Set the connecting flag
		c.mpdClientMutex.Lock()
		c.mpdClientConnecting = true
		c.mpdClientMutex.Unlock()

		// Notify the callback we're about to connect
		c.onStatusChange()

		// Try to connect
		log.Debugf("Connecting to MPD (network=%v, address=%v)", c.mpdNetwork, c.mpdAddress)
		if client, err = mpd.DialAuthenticated(c.mpdNetwork, c.mpdAddress, c.mpdPassword); err == nil {
			connected = true
		} else {
			err = errors.Errorf("DialAuthenticated() failed: %v", err)
		}
	}

	// If there's a local client, we've just connected
	status := mpd.Attrs{}
	if connected && client != nil {
		// Validate the connection by requesting MPD status and, on success, save the client connection
		if status, err = client.Status(); err == nil {
			c.mpdClientMutex.Lock()
			c.mpdClientConnecting = false
			c.mpdClient = client
			c.mpdClientMutex.Unlock()
			log.Info("Successfully connected to MPD")

			// Start the watcher
			go func() { c.chWatcherStart <- true }()
		} else {
			connected = false
			err = errors.Errorf("Status() after dial failed: %v", err)
			// Disconnect since we're not "fully connected"
			errCheck(client.Close(), "doConnect(): Close() failed")
		}

	} else {
		connected = false
		// We didn't connect. Validate the existing connection, if any
		c.IfConnected(func(client *mpd.Client) {
			wasConnected = true
			if status, err = client.Status(); err == nil {
				connected = true
			} else {
				err = errors.Errorf("Status() failed: %v", err)
			}
		})

		// Connection lost
		if wasConnected && !connected {
			log.Warning("Connection to MPD lost")

			// Remove client connection
			c.mpdClientMutex.Lock()
			c.mpdClientConnecting = false
			c.mpdClient = nil
			c.mpdClientMutex.Unlock()

			// Suspend the watcher
			go func() { c.chWatcherStop <- false }()
		}
	}

	// On error, replace status with the error info
	if errCheck(err, "Failed to connect to MPD") {
		status = mpd.Attrs{"error": err.Error()}
	}

	// Store the (updated) status
	c.setStatus(status)

	// Notify the status callback on status change
	if wasConnected != connected {
		c.onStatusChange()
	}

	if heartbeat {
		// No connection (anymore), re-attempt connection if needed, but not more frequently than once in a heartbeat
		if !connected && c.stayConnected {
			c.startConnecting()
		}

		// Notify the heartbeat callback
		c.onHeartbeat()
	}
}

// watch starts watching MPD subsystem changes
func (c *Connector) watch() {
	log.Debug("watch()")
	var rewatchTimer *time.Timer
	var eventChannel chan string
	var errorChannel chan error
	var mpdWatcher *mpd.Watcher
	for {
		select {
		// Request to watch
		case <-c.chWatcherStart:
			log.Debug("Start watcher")

			// Remove the timer
			rewatchTimer = nil

			// If no watcher yet
			if mpdWatcher == nil {
				watcher, err := mpd.NewWatcher(c.mpdNetwork, c.mpdAddress, c.mpdPassword)
				// Failed to connect
				if err != nil {
					log.Warning("Failed to watch MPD", err)
					// Schedule a reconnection
					rewatchTimer = time.AfterFunc(3*time.Second, func() {
						c.chWatcherStart <- true
					})

				} else {
					// Connection succeeded
					mpdWatcher = watcher
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
			c.setStatus(status)

			// Notify the callback
			c.onSubsystemChange(subsystem)

		// Watcher's error
		case err := <-errorChannel:
			log.Debug("Watcher error", err)

		// Request to quit
		case doQuit := <-c.chWatcherStop:
			// Kill the reconnection timer, if any
			if rewatchTimer != nil {
				rewatchTimer.Stop()
				rewatchTimer = nil
			}

			// Close the connection to MPD, if any
			if mpdWatcher != nil {
				log.Debug("Stop watcher")
				errCheck(mpdWatcher.Close(), "mpdWatcher.Close() failed")
				mpdWatcher = nil
			}

			// If we need to quit
			if doQuit {
				return
			}
		}
	}
}
