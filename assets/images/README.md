# Image Assets

This directory contains image files used by the Discord bot, primarily for onboarding guides and visual instructions.

## Directory Structure

```
assets/images/
├── onboarding/           # Onboarding session guide images
│   ├── step1.png        # Step 1: Welcome guide
│   ├── step2.png        # Step 2: Profile setup guide
│   ├── step4.png        # Step 4: Currency system guide
│   ├── step6-1.png      # Step 6: Member rank system (image 1)
│   └── step6-2.png      # Step 6: Level check method (image 2)
└── README.md            # This file

Note: Steps 3, 5, and 7 do not use images.
```

## Adding Images

1. **Place images** in the appropriate subdirectory (e.g., `onboarding/` for onboarding guides)
2. **Use PNG or JPG format** (PNG recommended for screenshots with text)
3. **Name files clearly** (e.g., `step1.png`, `step2.png`)
4. **Recommended size**: Discord embeds images well up to 2048x2048px, but 1000-1500px width is ideal for readability

## Using Images in Code

Images are loaded and sent as file attachments in Discord messages:

```go
// Example from onboarding_session.go
file, err := os.Open("assets/images/onboarding/step1.png")
if err != nil {
    s.logger.Error("failed to open image file", "error", err)
    return err
}
defer file.Close()

_, err = s.session.ChannelMessageSendComplex(s.vcChannelID, &discordgo.MessageSend{
    Content: "Guide instructions here",
    Files: []*discordgo.File{
        {
            Name:   "guide.png",
            Reader: file,
        },
    },
})
```

## Image Guidelines

- **Clear and readable**: Ensure text in images is legible at Discord's display size
- **Optimized file size**: Keep images under 8MB (Discord's limit)
- **Consistent style**: Use similar colors/formatting across guide images
- **Japanese text**: Use appropriate fonts that render well in screenshots

## Notes

- Images are bundled with the Docker container during build
- If images are updated, rebuild and redeploy the container
- Consider hosting large images externally (CDN) for faster updates without redeployment

