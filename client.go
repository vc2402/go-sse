package sse

// Client represents a web browser connection.
type Client struct {
	lastEventID,
	channel string
	name    string
	send    chan *Message
	version int
}

func newClient(lastEventID, channel string, name string, version int) *Client {
	return &Client{
		lastEventID,
		channel,
		name,
		make(chan *Message),
		version,
	}
}

// SendMessage sends a message to client.
func (c *Client) SendMessage(message *Message) {
	if message.version <= c.version {
		c.lastEventID = message.id
		c.send <- message
	}
}

// Channel returns the channel where this client is subscribe to.
func (c *Client) Channel() string {
	return c.channel
}

// LastEventID returns the ID of the last message sent.
func (c *Client) LastEventID() string {
	return c.lastEventID
}
