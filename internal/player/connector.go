package player

import (
	"github.com/fhs/gompd/v2/mpd"
	"github.com/yktoo/ymuse/internal/util"
	"sort"
	"strconv"
	"sync"
	"time"
)

// Connector encapsulates functionality for connecting to MPD and watch for its changes
type Connector struct {
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
	onConnected func()
	// Callback for periodic message notifications
	onHeartbeat func()
	// Callback for subsystem change notifications
	onSubsystemChange func(subsystem string)
}

func NewConnector(onConnected func(), onHeartbeat func(), onSubsystemChange func(subsystem string)) *Connector {
	return &Connector{
		mpdStatus:          mpd.Attrs{},
		chConnectorConnect: make(chan int),
		chConnectorQuit:    make(chan int),
		chWatcherStart:     make(chan int),
		chWatcherQuit:      make(chan int),
		onConnected:        onConnected,
		onHeartbeat:        onHeartbeat,
		onSubsystemChange:  onSubsystemChange,
	}
}

// Start() initialises the connector
func (c *Connector) Start() {
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
	close(c.chConnectorQuit)
	close(c.chWatcherQuit)
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

// PlayerPrevious() rewinds the player to the previous track
func (c *Connector) PlayerPrevious() {
	c.IfConnected(func(client *mpd.Client) {
		errCheck(client.Previous(), "Previous() failed")
	})
}

// PlayerStop() stops the playback
func (c *Connector) PlayerStop() {
	c.IfConnected(func(client *mpd.Client) {
		errCheck(client.Stop(), "Stop() failed")
	})
}

// PlayerPlayPause() pauses or resumes the playback
func (c *Connector) PlayerPlayPause() {
	c.IfConnected(func(client *mpd.Client) {
		switch c.Status()["state"] {
		case "pause":
			errCheck(client.Pause(false), "Pause(false) failed")
		case "play":
			errCheck(client.Pause(true), "Pause(true) failed")
		default:
			errCheck(client.Play(-1), "Play() failed")
		}
	})
}

// PlayerNext() advances the player to the next track
func (c *Connector) PlayerNext() {
	c.IfConnected(func(client *mpd.Client) {
		errCheck(client.Next(), "Next() failed")
	})
}

// PlayerToggleRandom() toggles player's random mode
func (c *Connector) PlayerToggleRandom() {
	c.IfConnected(func(client *mpd.Client) {
		errCheck(client.Random(c.Status()["random"] == "0"), "Random() failed")
	})
}

// PlayerToggleRepeat() toggles player's repeat mode
func (c *Connector) PlayerToggleRepeat() {
	c.IfConnected(func(client *mpd.Client) {
		errCheck(client.Repeat(c.Status()["repeat"] == "0"), "Repeat() failed")
	})
}

// PlayerToggleConsume() toggles player's consume mode
func (c *Connector) PlayerToggleConsume() {
	c.IfConnected(func(client *mpd.Client) {
		errCheck(client.Consume(c.Status()["consume"] == "0"), "Consume() failed")
	})
}

// QueueClear() empties MPD's play queue
func (c *Connector) QueueClear() {
	c.IfConnected(func(client *mpd.Client) {
		errCheck(client.Clear(), "Clear() failed")
	})
}

// QueueShuffle() randomises MPD's play queue
func (c *Connector) QueueShuffle() {
	c.IfConnected(func(client *mpd.Client) {
		errCheck(client.Shuffle(-1, -1), "Shuffle() failed")
	})
}

// QueueSort() orders MPD's play queue on the provided attribute
func (c *Connector) QueueSort(attrName string, numeric, descending bool) {
	c.IfConnected(func(client *mpd.Client) {
		// Fetch the current playlist
		attrs, err := client.PlaylistInfo(-1, -1)
		if errCheck(err, "PlaylistInfo() failed") {
			return
		}

		// Sort the list
		sort.SliceStable(attrs, func(i, j int) bool {
			a, b := attrs[i][attrName], attrs[j][attrName]
			if numeric {
				an, bn := util.ParseFloatDef(a, 0), util.ParseFloatDef(b, 0)
				if descending {
					return bn < an
				}
				return an < bn
			}
			if descending {
				return b < a
			}
			return a < b
		})

		// Post the changes back to MPD
		commands := client.BeginCommandList()
		for index, a := range attrs {
			id, err := strconv.Atoi(a["Id"])
			if errCheck(err, "ID attribute conversion to int failed") {
				return
			}
			commands.MoveID(id, index)
		}
		errCheck(commands.End(), "QueueSort: CommandList execution failed")

	})
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

			// If disconnected
			c.IfConnectedElse(
				nil,
				func() {
					// Try to connect to MPD
					cfg := util.GetConfig()
					client, err := mpd.DialAuthenticated("tcp", cfg.MpdAddress, cfg.MpdPassword)
					if errCheck(err, "Dial() failed") {
						return
					}

					// Connection succeeded
					c.mpdClient = client

					// Start the watcher
					go func() { c.chWatcherStart <- 1 }()

					// Actualise the status
					status, err := client.Status()
					c.mpdStatusMutex.Lock()
					if errCheck(err, "connect(): Status() failed") {
						c.mpdStatus = mpd.Attrs{}
					} else {
						c.mpdStatus = status
					}
					c.mpdStatusMutex.Unlock()

					// Notify the callback
					c.onConnected()
				})

		// Heartbeat tick
		case <-heartbeatTicker.C:
			c.IfConnectedElse(
				func(client *mpd.Client) {
					// Connection lost
					status, err := client.Status()
					if errCheck(err, "Status() failed: connection to MPD lost") {
						c.mpdClient = nil
						// Clear the status
						c.mpdStatusMutex.Lock()
						c.mpdStatus = mpd.Attrs{}
						c.mpdStatusMutex.Unlock()
						// Restart the connector goroutine
						c.startConnecting()

					} else {
						// Connection is okay, store the status
						c.mpdStatusMutex.Lock()
						c.mpdStatus = status
						c.mpdStatusMutex.Unlock()
					}
				},
				func() {
					c.mpdStatusMutex.Lock()
					c.mpdStatus = mpd.Attrs{}
					c.mpdStatusMutex.Unlock()
					c.startConnecting()
				})

			// Notify the callback
			c.onHeartbeat()

		// Request to quit
		case <-c.chConnectorQuit:
			// Kill the heartbeat timer
			heartbeatTicker.Stop()

			// Close the connection to MPD, if any
			c.IfConnected(func(client *mpd.Client) {
				log.Debug("Stop connector")
				errCheck(client.Close(), "Close() failed")
				c.mpdClient = nil
			})
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
				cfg := util.GetConfig()
				watcher, err := mpd.NewWatcher("tcp", cfg.MpdAddress, cfg.MpdPassword)
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
			c.mpdStatusMutex.Lock()
			c.mpdStatus = status
			c.mpdStatusMutex.Unlock()

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
