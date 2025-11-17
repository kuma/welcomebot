# Audio Playback Implementation Guide

## Overview

This document explains how audio playback is implemented in the welcomebot using the `github.com/jonas747/dca` library with Discord voice connections.

## Recommended Approach (from godoc)

Based on the official documentation, we use **`dca.StreamingSession`** which is the recommended way to play audio in Discord voice channels.

### Key Components

1. **`VoiceConnection`** (from discordgo)
   - Has `OpusSend` chan for sending audio
   - Has `Ready` bool to check if voice is ready

2. **`Decoder`** (from dca)
   - Reads DCA files
   - Implements `OpusReader` interface

3. **`StreamingSession`** (from dca)
   - Handles automatic frame transmission
   - Provides playback control (pause, resume)
   - Reports playback status

## Implementation

### Step 1: Join Voice Channel

```go
vc, err := session.ChannelVoiceJoin(guildID, channelID, false, true)
if err != nil {
    return fmt.Errorf("join voice: %w", err)
}

// Wait for connection to be ready
time.Sleep(250 * time.Millisecond)
```

### Step 2: Open and Decode DCA File

```go
file, err := os.Open("audio/kk/1-intro.dca")
if err != nil {
    return fmt.Errorf("open audio file: %w", err)
}
defer file.Close()

// Create decoder (implements OpusReader interface)
decoder := dca.NewDecoder(file)
```

### Step 3: Create Streaming Session

```go
// Create done channel to receive completion status
done := make(chan error)

// Create streaming session - handles sending frames automatically
stream := dca.NewStream(decoder, voiceConnection, done)
```

### Step 4: Wait for Playback Completion

```go
// Wait for playback to complete
select {
case err := <-done:
    if err != nil && err != io.EOF {
        return fmt.Errorf("playback error: %w", err)
    }
    // Playback completed successfully
case <-ctx.Done():
    stream.SetPaused(true) // Stop playback
    return fmt.Errorf("playback cancelled")
}
```

## Full Function

Here's our complete implementation:

```go
func (s *OnboardingSession) playAudioFile(guide, filename string) error {
    audioPath := fmt.Sprintf("audio/%s/%s", guide, filename)
    
    // Check if file exists
    if _, err := os.Stat(audioPath); os.IsNotExist(err) {
        return fmt.Errorf("audio file not found: %s", audioPath)
    }

    // Check if voice connection is ready
    if s.voiceConn == nil || !s.voiceConn.Ready {
        return fmt.Errorf("voice connection not ready")
    }

    // Open DCA file
    file, err := os.Open(audioPath)
    if err != nil {
        return fmt.Errorf("open audio file: %w", err)
    }
    defer file.Close()

    // Create decoder (implements OpusReader interface)
    decoder := dca.NewDecoder(file)
    
    // Create streaming session - this handles sending frames automatically
    done := make(chan error)
    stream := dca.NewStream(decoder, s.voiceConn, done)
    
    // Wait for playback to complete
    select {
    case err := <-done:
        if err != nil && err != io.EOF {
            return fmt.Errorf("playback error: %w", err)
        }
    case <-s.ctx.Done():
        stream.SetPaused(true) // Stop playback
        return fmt.Errorf("playback cancelled")
    }

    s.logger.Info("audio playback completed", "path", audioPath)
    return nil
}
```

## Advantages of StreamingSession

1. **Automatic Frame Transmission**: No need to manually read frames and send to `OpusSend` channel
2. **Playback Control**: Built-in pause/resume functionality
3. **Status Monitoring**: Can check playback position and completion
4. **Cleaner Code**: Less boilerplate, easier to maintain

## Alternative: Manual Frame Streaming

If you need more control, you can manually stream frames:

```go
decoder := dca.NewDecoder(file)

for {
    frame, err := decoder.OpusFrame()
    if err == io.EOF {
        break
    }
    if err != nil {
        return err
    }
    
    voiceConn.OpusSend <- frame
}
```

However, `StreamingSession` is the **recommended approach** as documented in godoc.

## Testing Audio Files

To test if a DCA file is valid:

```bash
# Convert back to WAV for testing
ffmpeg -f s16le -ar 48000 -ac 2 -i <(cat audio/kk/1-intro.dca) test.wav

# Play on macOS
afplay test.wav
```

## References

- **Package godoc**: `go doc github.com/jonas747/dca`
- **StreamingSession**: `go doc github.com/jonas747/dca.StreamingSession`
- **Decoder**: `go doc github.com/jonas747/dca.Decoder`
- **VoiceConnection**: `go doc github.com/bwmarrin/discordgo.VoiceConnection`

## Example Usage in Onboarding Flow

```go
// After guide is selected, play tutorial steps
for step := 1; step <= 7; step++ {
    filename := fmt.Sprintf("%d-*.dca", step)
    
    if err := s.playAudioFile(selectedGuide, filename); err != nil {
        s.logger.Error("failed to play audio", "step", step, "error", err)
        continue
    }
    
    // Show interactive buttons between steps
    s.sendStepMessage(step)
    
    // Wait for user interaction
    // ...
}
```

## Troubleshooting

### Audio Not Playing
1. Check `voiceConn.Ready` is `true`
2. Verify DCA file exists and is valid
3. Ensure bot has `Speak` permission in voice channel
4. Check logs for frame transmission errors

### Audio Quality Issues
- DCA files should be 48kHz, Opus-encoded
- Use 128kbps bitrate for high quality
- Convert from high-quality source files

### Playback Stuttering
- Check network latency
- Verify voice connection stability
- Monitor CPU usage during playback

