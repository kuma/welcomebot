# Audio Files for Welcome Onboarding

This directory contains audio files used during the voice onboarding process.

## Required Files

Place your audio files here with the following names:

- `welcome.mp3` - Initial welcome message when user joins onboarding VC
- `step1.mp3` - Audio for step 1
- `step2.mp3` - Audio for step 2
- `voice_a.mp3` - Audio for voice option A
- `voice_b.mp3` - Audio for voice option B
- `completion.mp3` - Completion message

## Audio Format

- **Format**: MP3 or WAV
- **Bitrate**: 128kbps or higher recommended
- **Sample Rate**: 48kHz (Discord standard)
- **Channels**: Mono or Stereo

## Audio Encoding for Discord

Discord voice requires DCA (Discord Compatible Audio) format. The bot will need to:

1. Read the audio file
2. Convert to DCA format
3. Stream to voice connection

### Example using `dca` package

```go
import "github.com/jonas747/dca"

// Encode audio file to DCA
options := dca.StdEncodeOptions
options.RawOutput = true
options.Bitrate = 128

encodeSession, err := dca.EncodeFile("./audio/welcome.mp3", options)
if err != nil {
    return err
}
defer encodeSession.Cleanup()

// Stream to Discord
done := make(chan error)
dca.NewStream(encodeSession, voiceConnection, done)
err := <-done
```

## Creating Audio Files

### Text-to-Speech Options

1. **Google Cloud Text-to-Speech**
   ```bash
   curl -X POST \
     -H "Authorization: Bearer $(gcloud auth print-access-token)" \
     -H "Content-Type: application/json" \
     -d '{
       "input": {"text": "Welcome to the server!"},
       "voice": {"languageCode": "en-US", "name": "en-US-Neural2-C"},
       "audioConfig": {"audioEncoding": "MP3"}
     }' \
     "https://texttospeech.googleapis.com/v1/text:synthesize" \
     > welcome.mp3
   ```

2. **Amazon Polly**
   ```bash
   aws polly synthesize-speech \
     --output-format mp3 \
     --voice-id Joanna \
     --text "Welcome to the server!" \
     welcome.mp3
   ```

3. **Local TTS (eSpeak)**
   ```bash
   espeak "Welcome to the server!" --stdout | \
     ffmpeg -i - -ar 48000 -ac 2 -b:a 128k welcome.mp3
   ```

### Recording Your Own Audio

1. Record with any audio software (Audacity, Adobe Audition, etc.)
2. Export as MP3, 48kHz, 128kbps
3. Keep recordings concise (under 30 seconds per file)
4. Use clear, friendly voice
5. Add slight echo/reverb for professional sound

## Placeholder Files

Until you create your actual audio files, you can use silence:

```bash
# Create 5 seconds of silence as placeholder
ffmpeg -f lavfi -i anullsrc=r=48000:cl=mono -t 5 -q:a 9 -acodec libmp3lame welcome.mp3
```

## Future: Dynamic Audio

In future versions, the bot will support:
- Per-guild custom audio
- Text-to-speech generation on-the-fly
- Multi-language audio files
- Admin upload via Discord

## Notes

- Audio files are not included in git (add to .gitignore)
- Each server admin should provide their own audio files
- Keep file sizes reasonable (< 5MB per file)
- Test audio in Discord before production use

