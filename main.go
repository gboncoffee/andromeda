package main

import (
    "fmt"
    "os"
    "os/signal"
    "syscall"
    "strings"
    "time"
)

import dc "github.com/bwmarrin/discordgo"

// basic commands {{{
func ping(s *dc.Session, m *dc.MessageCreate) {
    s.ChannelMessageSend(m.ChannelID, "pong!")
}

func echo(s *dc.Session, m *dc.MessageCreate, command []string) {
    msg := ""
    for i, c := range command {
        if 0 == i {
            continue
        }
        msg += " "
        msg += c
    }
    s.ChannelMessageSend(m.ChannelID, msg)
}

func unknown_command(s *dc.Session, m *dc.MessageCreate, command string) {
    s.ChannelMessageSend(m.ChannelID, "Unknown command: " + command)
}
// }}}

func userIsOnChannel(user *string, channel *dc.Channel) bool {
    for _, u := range channel.Members {
        if u.ID == *user {
            return true
        }
    }
    return false
}

func getUserVoiceChannel(user *string, channels []*dc.Channel) *dc.Channel {
    for _, channel := range channels {
        if dc.ChannelTypeGuildVoice == channel.Type {
            // if userIsOnChannel(user, channel) {
            return channel
            // }
        }
    }
    return nil
}

func join(s *dc.Session, m *dc.MessageCreate) {
    author     := m.Author.ID
    guild, err := s.State.Guild(m.GuildID)
    // wtf no guild for Andromeda?
    if nil != err {
        return
    }
    channel := getUserVoiceChannel(&author, guild.Channels)

    if nil == channel {
        s.ChannelMessageSend(m.ChannelID, "You're not connect to any voice channel!")
        return
    }

    vc, err := s.ChannelVoiceJoin(m.GuildID, channel.ID, false, true)
    if nil != err {
        s.ChannelMessageSend(m.ChannelID, "Cannot connect to your voice channel!")
        return
    }

    start := time.Now()
    end   := time.Now()
    for 5000 > end.Sub(start) {
        if vc.Ready {
            break
        }
        end = time.Now()
    }
    if !vc.Ready {
        s.ChannelMessageSend(m.ChannelID, "Connection to voice channel timeouted.")
        return
    }
}

// connection stuff {{{
func connectToDiscord(token string) *dc.Session {
    session, err := dc.New("Bot " + token)
    if nil != err {
        panic(fmt.Sprintf("Cannot connect to Discord. Bailing out..."))
    }
    return session
}
// }}}

// message create {{{
func messageCreate(s *dc.Session, m *dc.MessageCreate) {

    // ignore messages by the bot itself
    if m.Author.ID == s.State.User.ID {
        return
    }

    // ignore empty messages
    if 0 == len(m.Content) {
        return
    }

    // ignore messages that arent commands
    if '!' != m.Content[0] || 1 == len(m.Content) {
        return
    }

    command := strings.Fields(m.Content[1:])

    switch command[0] {
    case "ping":
        ping(s, m)
    case "echo":
        echo(s, m, command)
    case "join":
        join(s, m)
    default:
        unknown_command(s, m, command[0])
    }
}
// }}}

func main() {
    token, exists := os.LookupEnv("DISCORD_TOKEN")
    if !exists {
        panic(fmt.Sprintf("DISCORD_TOKEN is not set. Bailing out..."))
    }
    session := connectToDiscord(token)

    session.AddHandler(messageCreate)
    session.Identify.Intents = dc.IntentsAll

    err := session.Open()
    if nil != err {
        panic(fmt.Sprintf("Cannot open WebSocket. Bailing out..."))
    }

    // wait for a signal to bail out
    fmt.Println("Bot is now running.  Press CTRL-C to exit.")
    sc := make(chan os.Signal, 1)
    signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
    <-sc

    session.Close()
}
