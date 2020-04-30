package player

import (
	"github.com/fhs/gompd/v2/mpd"
	"sync"
	"time"
)

// Connector encapsulates functionality for connecting to MPD and watch for its changes
type Connector struct {
	// MPD's IP address
	mpdAddress string

	// MPD client instance
	mpdClient      *mpd.Client
	mpdClientMutex sync.Mutex

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
	onConnected func()
	// Callback for periodic message notifications
	onHeartbeat func()
	// Callback for subsystem change notifications
	onSubsystemChange func(subsystem string)
}

func NewConnector(mpdAddress string, onConnected func(), onHeartbeat func(), onSubsystemChange func(subsystem string)) *Connector {
	return &Connector{
		mpdAddress:         mpdAddress,
		chConnectorConnect: make(chan int),
		chConnectorQuit:    make(chan int),
		chWatcherStart:     make(chan int),
		chWatcherQuit:      make(chan int),
		onConnected:        onConnected,
		onHeartbeat:        onHeartbeat,
		onSubsystemChange:  onSubsystemChange,
	}
}

// Start() signals the connector to initiate connection process
func (c *Connector) Start() {
	// Start the connect goroutine
	go c.connect()

	// Start the watch goroutine
	go c.watch()

	// Signal the connector to start
	go func() { c.chConnectorConnect <- 1 }()
}

// Stop() signals the connector to shut down
func (c *Connector) Stop() {
	close(c.chConnectorQuit)
	close(c.chWatcherQuit)
}

// IfConnected() runs MPD client code if there's a connection with MPD and/or code if there's no connection
func (c *Connector) IfConnected(funcIfConnected func(client *mpd.Client), funcIfDisconnected func()) {
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

// connect() takes care of establishing a connection to MPD
func (c *Connector) connect() {
	log.Debug("connect()")
	var heartbeatTicker = time.NewTicker(time.Second)
	for {
		select {
		// Request to connect
		case <-c.chConnectorConnect:
			log.Debug("Start connector")

			// If disconnected
			c.IfConnected(
				nil,
				func() {
					// Try to connect to MPD
					client, err := mpd.Dial("tcp", c.mpdAddress)
					if err != nil {
						errCheck(err, "Dial() failed")
						return
					}

					// Connection succeeded
					c.mpdClient = client

					// Start the watcher
					go func() { c.chWatcherStart <- 1 }()

					// Notify the callback
					c.onConnected()
				})

		// Heartbeat tick
		case <-heartbeatTicker.C:
			c.IfConnected(
				func(client *mpd.Client) {
					// Connection lost
					if err := client.Ping(); err != nil {
						log.Debug("Ping(): connection to MPD lost", err)
						c.mpdClient = nil
						c.Start()
					}
				},
				func() {
					c.Start()
				})

			// Notify the callback
			c.onHeartbeat()

		// Request to quit
		case <-c.chConnectorQuit:
			// Kill the heartbeat timer
			heartbeatTicker.Stop()

			// Close the connection to MPD, if any
			c.IfConnected(
				func(client *mpd.Client) {
					log.Debug("Stop connector")
					errCheck(client.Close(), "Close() failed")
					c.mpdClient = nil
				},
				nil)
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
				watcher, err := mpd.NewWatcher("tcp", c.mpdAddress, "")
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
