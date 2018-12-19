package sse

// Channel represents a server sent events channel.
type Channel struct {
	lastEventID,
	name string
	clients map[string]*Client
}

func newChannel(name string) *Channel {
	return &Channel{
		"",
		name,
		make(map[string]*Client),
	}
}

// SendMessage broadcast a message to all clients in a channel.
func (c *Channel) SendMessage(name string, message *Message) {
	if name != "" {
		if cl, ok := c.clients[name]; ok {
			cl.SendMessage(message)
		}
	} else {
		c.lastEventID = message.id

		for _, cl := range c.clients {
			if cl != nil {
				cl.SendMessage(message)
			}
		}
	}
}

// Close closes the channel and disconnect all clients.
func (c *Channel) Close() {
	// Kick all clients of this channel.
	for _, client := range c.clients {
		c.removeClient(client)
	}
}

// ClientCount returns the number of clients connected to this channel.
func (c *Channel) ClientCount() int {
	return len(c.clients)
}

// LastEventID returns the ID of the last message sent.
func (c *Channel) LastEventID() string {
	return c.lastEventID
}

func (c *Channel) addClient(client *Client) {
	c.clients[client.name] = client
}

func (c *Channel) removeClient(client *Client) {
	c.clients[client.name] = nil
	close(client.send)
	delete(c.clients, client.name)
}
