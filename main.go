package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"encoding/binary"
	"io"

	"github.com/bwmarrin/discordgo"
)

var voice *discordgo.VoiceConnection
var buffer = make([][]byte, 0)

// This was basically copy-pasted from
// https://github.com/bwmarrin/discordgo/blob/master/examples/airhorn/main.go
func Load(dir string, name string) error {
	if strings.ContainsAny(dir, "/") || strings.ContainsAny(name, "/") || strings.HasPrefix(dir, ".") || strings.HasPrefix(name, ".") {
		return errors.New("someone is trying to hack lmao")
	}

	file, err := os.Open(dir + "/" + name + ".dca")
	if err != nil {
		return err
	}

	var opuslen int16

	for {
		// Read opus frame length from dca file.
		err = binary.Read(file, binary.LittleEndian, &opuslen)

		// If this is the end of the file, just return.
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			err := file.Close()
			if err != nil {
				return err
			}
			return nil
		}

		if err != nil {
			return err
		}

		// Read encoded pcm from dca file.
		InBuf := make([]byte, opuslen)
		err = binary.Read(file, binary.LittleEndian, &InBuf)

		// Should not be any end of file errors.
		if err != nil {
			return err
		}

		// Append encoded pcm data to the buffer.
		buffer = append(buffer, InBuf)
	}
}

func Play(dir string, name string) error {
	err := Load(dir, name)
	if err != nil {
		return err
	}

	go func() {
		voice.Speaking(true)

		for _, buff := range buffer {
			voice.OpusSend <- buff
		}

		voice.Speaking(false)
	}()

	return nil
}

func DispatchCommand(discord *discordgo.Session, command string, args []string, message *discordgo.Message) {
	switch command {
	case "ping":
		discord.ChannelMessageSend(message.ChannelID, "pong!")
	case "join":
		guild, _ := discord.State.Guild(message.GuildID)
		for _, voiceState := range guild.VoiceStates {
			if voiceState.UserID == message.Author.ID {
				voice, _ = discord.ChannelVoiceJoin(message.GuildID, voiceState.ChannelID, false, true)
				return
			}
		}
	case "leave":
		if voice != nil {
			voice.Disconnect()
			voice = nil
		}
	case "play":
		if voice != nil && len(args) >= 2 {
			err := Play(args[0], args[1])
			if err != nil {
				discord.ChannelMessageSend(message.ChannelID, "i cant lmaofdsfe")
			}
		} else {
			discord.ChannelMessageSend(message.ChannelID, "play what? where? mf")
		}
	}
}

func main() {
	token := os.Getenv("DISCORD_TOKEN")
	discord, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatalf("Could not login to Discord: %v\n", err)
	}

	discord.AddHandler(func(discord *discordgo.Session, event *discordgo.MessageCreate) {
		if event.Author.ID == discord.State.User.ID || !strings.HasPrefix(event.Content, "!") {
			return
		}

		command, args, _ := strings.Cut(event.Content[1:], " ")
		DispatchCommand(discord, command, strings.Split(args, " "), event.Message)
	})

	discord.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMessages | discordgo.IntentsGuildVoiceStates

	err = discord.Open()
	if err != nil {
		log.Fatalf("Couldn't open websocket with Discord: %v", err)
	}

	// Keep running waiting for actions on the websocket or signals in the
	// system.
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-signals

	discord.Close()

	// Feed a line in the terminal just to don't mess up the prompt.
	fmt.Println()
}
