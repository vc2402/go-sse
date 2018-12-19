package sse

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
)

// Server represents a server sent events server.
type Server struct {
	options      *Options
	channels     map[string]*Channel
	addClient    chan *Client
	removeClient chan *Client
	shutdown     chan bool
	closeChannel chan string
}

// NewServer creates a new SSE server.
func NewServer(options *Options) *Server {
	if options == nil {
		options = &Options{
			Logger: log.New(os.Stdout, "go-sse: ", log.LstdFlags),
		}
	}

	if options.Logger == nil {
		options.Logger = log.New(ioutil.Discard, "", log.LstdFlags)
	}

	s := &Server{
		options,
		make(map[string]*Channel),
		make(chan *Client),
		make(chan *Client),
		make(chan bool),
		make(chan string),
	}

	go s.dispatch()

	return s
}

func (s *Server) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	flusher, ok := response.(http.Flusher)

	if !ok {
		http.Error(response, "Streaming unsupported.", http.StatusInternalServerError)
		return
	}

	h := response.Header()

	if s.options.hasHeaders() {
		for k, v := range s.options.Headers {
			h.Set(k, v)
		}
	}

	if request.Method == "GET" {
		h.Set("Content-Type", "text/event-stream")
		h.Set("Cache-Control", "no-cache")
		h.Set("Connection", "keep-alive")

		var channelName string
		var clientName string
		var version int

		if s.options.ChannelNameFunc == nil {
			channelName = request.URL.Path
		} else {
			channelName, clientName, version = s.options.ChannelNameFunc(request)
		}

		lastEventID := request.Header.Get("Last-Event-ID")
		c := newClient(lastEventID, channelName, clientName, version)
		s.addClient <- c
		closeNotify := response.(http.CloseNotifier).CloseNotify()

		go func() {
			<-closeNotify
			s.removeClient <- c
			if s.options.OnClientDisconnectFunc != nil {
				s.options.OnClientDisconnectFunc(c.channel, c.name)
			}
		}()

		response.WriteHeader(http.StatusOK)
		flusher.Flush()

		for msg := range c.send {
			msg.retry = s.options.RetryInterval
			fmt.Fprintf(response, msg.String())
			flusher.Flush()
		}
	} else if request.Method != "OPTIONS" {
		response.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// SendMessage broadcast a message to all clients in a channel.
// If channel is an empty string, it will broadcast the message to all channels.
func (s *Server) SendMessage(channel string, message *Message) {
	s.SendMessageToClient(channel, "", message)
}

// SendMessageToClient send a message to specified client in a channel.
// If channel is an empty string, it will broadcast the message to all channels.
// If client is empty string, it will broadcast message to all clients in the channel
func (s *Server) SendMessageToClient(channel string, client string, message *Message) {
	if len(channel) == 0 {
		if message.event != "heartbeat" {
			s.options.Logger.Print("broadcasting message to all channels.")
		}

		for _, ch := range s.channels {
			ch.SendMessage(client, message)
		}
	} else if _, ok := s.channels[channel]; ok {
		s.options.Logger.Printf("message sent to channel '%s'.", channel)
		s.channels[channel].SendMessage(client, message)
	} else {
		s.options.Logger.Printf("message not sent because channel '%s' has no clients.", channel)
	}
}

// Restart closes all channels and clients and allow new connections.
func (s *Server) Restart() {
	s.options.Logger.Print("restarting server.")

	s.close()
}

// Shutdown performs a graceful server shutdown.
func (s *Server) Shutdown() {
	s.shutdown <- true
}

// ClientCount returns the number of clients connected to this server.
func (s *Server) ClientCount() int {
	i := 0

	for _, channel := range s.channels {
		i += channel.ClientCount()
	}

	return i
}

// HasChannel returns true if the channel associated with name exists.
func (s *Server) HasChannel(name string) bool {
	_, ok := s.channels[name]
	return ok
}

// GetChannel returns the channel associated with name or nil if not found.
func (s *Server) GetChannel(name string) (*Channel, bool) {
	ch, ok := s.channels[name]
	return ch, ok
}

// Channels returns a list of all channels to the server.
func (s *Server) Channels() []string {
	channels := []string{}

	for name := range s.channels {
		channels = append(channels, name)
	}

	return channels
}

// CloseChannel closes a channel.
func (s *Server) CloseChannel(name string) {
	s.closeChannel <- name
}

func (s *Server) close() {
	for name := range s.channels {
		s.closeChannel <- name
	}
}

func (s *Server) dispatch() {
	s.options.Logger.Print("server started.")

	for {
		select {

		// New client connected.
		case c := <-s.addClient:
			ch, exists := s.channels[c.channel]

			if !exists {
				ch = newChannel(c.channel)
				s.channels[ch.name] = ch

				s.options.Logger.Printf("channel '%s' created.", ch.name)
			}

			ch.addClient(c)
			s.options.Logger.Printf("new client connected to channel '%s'.", ch.name)

		// Client disconnected.
		case c := <-s.removeClient:
			if ch, exists := s.channels[c.channel]; exists {
				ch.removeClient(c)
				s.options.Logger.Printf("client disconnected from channel '%s'.", ch.name)

				s.options.Logger.Printf("checking if channel '%s' has clients.", ch.name)
				if ch.ClientCount() == 0 {
					delete(s.channels, ch.name)
					ch.Close()

					s.options.Logger.Printf("channel '%s' has no clients.", ch.name)
				}
			}

		// Close channel and all clients in it.
		case channel := <-s.closeChannel:
			if ch, exists := s.channels[channel]; exists {
				delete(s.channels, channel)
				ch.Close()
				s.options.Logger.Printf("channel '%s' closed.", ch.name)
			} else {
				s.options.Logger.Printf("requested to close channel '%s', but it doesn't exists.", channel)
			}

		// Event Source shutdown.
		case <-s.shutdown:
			s.close()
			close(s.addClient)
			close(s.removeClient)
			close(s.closeChannel)
			close(s.shutdown)

			s.options.Logger.Print("server stopped.")
			return
		case <-time.After(15 * time.Second):
			if s.options.Heartbeat {
				s.SendMessage("", "", &Message{event: "heartbeat"})
			}
		}
	}
}
