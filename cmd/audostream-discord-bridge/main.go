package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	"github.com/pion/rtp"

	"github.com/stieneee/mumble-discord-bridge/internal/stream"
)

const (
	defaultSampleRate   = 48000
	defaultChannelCount = 2
)

var (
	version string
	commit  string
	date    string
)

func main() {
	fmt.Println("AudioStream Discord Bridge")
	fmt.Println(version + " " + commit + " " + date)

	if err := godotenv.Load(); err != nil {
		log.Println("Failed to load .env file:", err)
	}

	discordToken := flag.String("discord-token", lookupEnvOrString("DISCORD_TOKEN", ""), "DISCORD_TOKEN, discord bot token, required")
	discordGID := flag.String("discord-gid", lookupEnvOrString("DISCORD_GID", ""), "DISCORD_GID, discord guild id, required")
	discordCID := flag.String("discord-cid", lookupEnvOrString("DISCORD_CID", ""), "DISCORD_CID, discord channel id, required")
	httpBind := flag.String("http-bind", lookupEnvOrString("HTTP_BIND", ":8080"), "HTTP_BIND, address for the HTTP audio server")
	streamBuffer := flag.Int("stream-buffer", lookupEnvOrInt("STREAM_BUFFER", 256), "STREAM_BUFFER, per listener packet buffer size")

	flag.Parse()
	log.Printf("app.config %v\n", getConfig(flag.CommandLine))

	if *discordToken == "" {
		log.Fatalln("missing discord bot token")
	}
	if *discordGID == "" {
		log.Fatalln("missing discord guild id")
	}
	if *discordCID == "" {
		log.Fatalln("missing discord channel id")
	}

	hub := stream.NewHub()
	mux := http.NewServeMux()
	mux.Handle("/stream", stream.NewHandler(hub, defaultSampleRate, defaultChannelCount, *streamBuffer))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	server := &http.Server{
		Addr:    *httpBind,
		Handler: mux,
	}

	serverErr := make(chan error, 1)
	go func() {
		log.Printf("HTTP audio server listening on %s", *httpBind)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	session, err := discordgo.New("Bot " + *discordToken)
	if err != nil {
		log.Fatalf("Failed to create Discord session: %v", err)
	}

	session.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsGuilds | discordgo.IntentsGuildVoiceStates)

	if err := session.Open(); err != nil {
		log.Fatalf("Failed to open Discord session: %v", err)
	}
	defer func() {
		if err := session.Close(); err != nil {
			log.Printf("Error closing Discord session: %v", err)
		}
	}()

	voice, err := session.ChannelVoiceJoin(*discordGID, *discordCID, false, false)
	if err != nil {
		log.Fatalf("Failed to join Discord voice channel: %v", err)
	}
	defer func() {
		if err := voice.Disconnect(); err != nil {
			log.Printf("Error disconnecting voice connection: %v", err)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go forwardAudio(ctx, voice, hub)

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)

	select {
	case sig := <-signals:
		log.Printf("Received signal %s, shutting down", sig)
	case err := <-serverErr:
		if err != nil {
			log.Printf("HTTP server stopped with error: %v", err)
		}
	}

	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Error shutting down HTTP server: %v", err)
	}
}

func forwardAudio(ctx context.Context, voice *discordgo.VoiceConnection, hub *stream.Hub) {
	logger := log.New(os.Stdout, "", log.LstdFlags)
	logger.Println("Waiting for Discord voice connection to become ready")

	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		voice.RLock()
		ready := voice.Ready && voice.OpusRecv != nil
		voice.RUnlock()

		if ready {
			logger.Println("Discord voice connection ready, streaming audio")
			break
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}

	for {
		select {
		case <-ctx.Done():
			logger.Println("Audio forwarder context cancelled")
			return
		case packet, ok := <-voice.OpusRecv:
			if !ok {
				logger.Println("Discord voice receive channel closed")
				return
			}
			if len(packet.Opus) == 0 {
				continue
			}

			payload := make([]byte, len(packet.Opus))
			copy(payload, packet.Opus)

			rtpPacket := &rtp.Packet{
				Header: rtp.Header{
					Version:        2,
					PayloadType:    111,
					SequenceNumber: packet.Sequence,
					Timestamp:      packet.Timestamp,
					SSRC:           packet.SSRC,
				},
				Payload: payload,
			}

			hub.Broadcast(rtpPacket)
		}
	}
}
