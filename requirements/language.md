# Feature: Language Configuration

## User-Facing Description
Configure the bot's language for your server. Supports English and Japanese.

## Commands/Interactions
- Accessible via `/menu` → "🌐 Set Language" button
- Shows buttons for language selection
- Admin only

## Flow
1. User (admin) clicks "🌐 Set Language" in menu
2. Bot shows language selection buttons: [English] [日本語]
3. User clicks their preferred language
4. Bot saves preference to database
5. Bot confirms in the selected language

## Data Models

### Database
- Uses existing `guild_languages` table
- `guild_id` (PK), `language_code` (en/ja)

### Cache
- Key: `welcomebot:i18n:guild:{guild_id}`
- TTL: Indefinite (cached until changed)

## Business Logic
- Admin permission required (Discord admin OR "welcomebotbotadmin" OR custom role)
- Per-guild configuration
- Immediate effect (cache updated)
- Bilingual display before selection

## Examples

### Example 1: Setting Language
```
User: [Clicks "🌐 Set Language" in menu]
Bot: "Choose your language / 言語を選択"
     [English] [日本語]
User: [Clicks 日本語]
Bot: "✅ 言語を日本語に設定しました"
```

### Example 2: Changing Language
```
User: [Clicks "🌐 Set Language" again]
Bot: "現在の言語: 日本語"
     [English] [日本語]
User: [Clicks English]
Bot: "✅ Language set to English"
```

## Technical Requirements
- Guild-aware (filter by guild_id)
- Uses i18n service for post-selection messages
- Shows bilingual text pre-selection
- Updates cache after save
- Returns to menu after completion (optional)

