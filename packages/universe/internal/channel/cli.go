package channel

import (
	"context"
	"fmt"
)

// ChannelMessage represents a message sent through a channel.
type ChannelMessage struct {
	Content string
	Source  string
}

// CLIChannel implements the Channel port for direct CLI interaction.
type CLIChannel struct{}

// NewCLI creates a new CLIChannel adapter.
func NewCLI() *CLIChannel {
	return &CLIChannel{}
}

// Name returns the channel identifier.
func (c *CLIChannel) Name() string {
	return "cli"
}

// Send outputs a message to the CLI.
func (c *CLIChannel) Send(ctx context.Context, msg ChannelMessage) error {
	fmt.Println(msg.Content)
	return nil
}

// Receive is not supported for the CLI channel.
func (c *CLIChannel) Receive(ctx context.Context) (<-chan ChannelMessage, error) {
	return nil, fmt.Errorf("cli channel does not support async receive.\nUse a different channel type for async messaging")
}

// Close is a no-op for the CLI channel.
func (c *CLIChannel) Close() error {
	return nil
}
