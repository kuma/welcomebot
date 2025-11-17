# Image Usage Guide for Onboarding

## Quick Start

### 1. Add Your Images

Place your guide images in the `assets/images/onboarding/` directory:

```
assets/images/onboarding/
├── step1.png   # Welcome guide
├── step2.png   # Profile setup guide
├── step3.png   # Role selection guide
├── step4.png   # Point system guide
├── step5.png   # Club information guide
├── step6.png   # Membership guide
└── step7.png   # Final instructions
```

### 2. Recommended Image Specs

- **Format**: PNG (for screenshots with text) or JPG
- **Width**: 1000-1500px (optimal for Discord)
- **File size**: Under 8MB (Discord limit)
- **DPI**: 72-96 DPI is sufficient for screen display

### 3. How Images Are Sent

When a user reaches a step, the bot will:
1. Send the embed with text and buttons
2. Send the guide image as an attachment (if available)
3. Play the audio guide

**Example for Step 1:**
```
[Embed]
🎬 BunnyClubへようこそ
[Text content here]
[次へ] [もう一度聞く]

[Image Attachment]
step1.png
```

### 4. Code Integration

Images are sent using the helper method `sendGuideImage()`:

```go
// In StartStep1, StartStep2, etc.
if err := s.sendGuideImage("step1.png"); err != nil {
    s.logger.Warn("failed to send step 1 guide image", "error", err)
}
```

The method:
- Automatically looks in `assets/images/onboarding/`
- Gracefully handles missing images (won't crash the bot)
- Logs warnings if images are not found
- Sends the image as a file attachment

### 5. Adding Images to Other Steps

To add images to Step 3, 4, 5, 6, or 7, use the same pattern:

```go
// In StartStep3()
if err := s.sendGuideImage("step3.png"); err != nil {
    s.logger.Warn("failed to send step 3 guide image", "error", err)
}
```

### 6. Deployment

Images are included in the Docker container automatically:
- The Dockerfile copies `assets/` directory
- Rebuild and push the container: `./scripts/prod-reload.sh`
- Images will be available to all pods

### 7. Testing Locally

Before deploying, test locally:

1. Add your image to `assets/images/onboarding/step1.png`
2. Run the bot locally: `go run ./cmd/worker/...`
3. Trigger the onboarding flow
4. Verify the image appears after the text embed

### 8. Best Practices

✅ **DO:**
- Use clear, high-contrast images
- Annotate images with arrows/numbers to guide users
- Keep file sizes reasonable (<2MB per image)
- Use consistent styling across all guide images
- Test on mobile Discord to ensure readability

❌ **DON'T:**
- Use extremely large images (>2MB) unless necessary
- Include sensitive information in screenshots
- Rely solely on images (always provide text instructions too)

### 9. Updating Images

To update an image after deployment:
1. Replace the image file in `assets/images/onboarding/`
2. Run `./scripts/prod-reload.sh` to rebuild and push new container
3. Kubernetes will roll out the update automatically

### 10. Troubleshooting

**Image not showing:**
- Check file exists: `ls assets/images/onboarding/step1.png`
- Check logs: `./scripts/prod-logs.sh` and look for "guide image not found"
- Verify Dockerfile copies assets: `COPY --from=builder /build/assets /app/assets`

**Image too large:**
- Optimize with tools like `pngquant` or `imagemagick`
- Resize to 1200px width max
- Convert to WebP for better compression

**Wrong image displaying:**
- Check filename matches exactly (case-sensitive)
- Clear browser cache in Discord (Ctrl+Shift+R)
- Verify correct image was uploaded

## Example Workflow

```bash
# 1. Create your guide image (using Photoshop, Figma, etc.)
# 2. Save as step1.png

# 3. Copy to the project
cp ~/Desktop/bunnyclub-guide-step1.png assets/images/onboarding/step1.png

# 4. Test locally
go run ./cmd/worker/main.go onboarding_handlers.go

# 5. Deploy to production
./scripts/prod-reload.sh

# 6. Monitor logs
./scripts/prod-logs.sh
# Select worker pod and verify "guide image sent successfully"
```

## Current Integration Status

✅ **Step 1** - Embed with image support (`step1.png`)
✅ **Step 2** - Text-Image-Text pattern (`step2.png`)
✅ **Step 3** - Plain markdown intro (no image), embeds for sub-steps
✅ **Step 4** - Text-Image-Text pattern (`step4.png`)
✅ **Step 5** - Plain markdown (no image)
✅ **Step 6** - Text-Image-Text-Image pattern (`step6-1.png`, `step6-2.png`)
✅ **Step 7** - Plain markdown with completion button (no image)

### Message Patterns Used

**Pattern 1: Embed with Image (Step 1)**
- Embed with description and buttons
- Separate image message

**Pattern 2: Text-Image-Text (Steps 2, 4)**
- Message 1: Plain markdown text (part 1)
- Message 2: Image attachment
- Message 3: Plain markdown text (part 2) with buttons

**Pattern 3: Plain Markdown (Steps 3, 5)**
- Plain markdown intro text
- Step 3 sub-steps use embeds with buttons

**Pattern 4: Text-Image-Text-Image (Step 6)**
- Message 1: Plain markdown text (part 1)
- Message 2: First image attachment
- Message 3: Plain markdown text (part 2)
- Message 4: Second image attachment
- Message 5: Buttons only

