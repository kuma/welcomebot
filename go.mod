module welcomebot

go 1.24

require (
	github.com/bwmarrin/discordgo v0.29.0
	github.com/go-redis/redis/v8 v8.11.5
	github.com/jonas747/dca v0.0.0-20210930103944-155f5e5f0cc7
	github.com/lib/pq v1.10.9
	github.com/sirupsen/logrus v1.9.3
)

// Use patched discordgo with new voice encryption modes (PR #1593)
// https://github.com/bwmarrin/discordgo/pull/1593
replace github.com/bwmarrin/discordgo => github.com/ozraru/discordgo v0.26.2-0.20251101184423-6792228f3271

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/gorilla/websocket v1.5.4-0.20250319132907-e064f32e3674 // indirect
	github.com/jonas747/ogg v0.0.0-20161220051205-b4f6f4cf3757 // indirect
	golang.org/x/crypto v0.40.0 // indirect
	golang.org/x/net v0.41.0 // indirect
	golang.org/x/sys v0.34.0 // indirect
)
