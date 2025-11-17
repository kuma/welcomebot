# Feature: Bot Info Command

## User-Facing Description
Displays information about the bot including version, uptime, and server count.

## Commands/Interactions
- `/botinfo`: Shows bot information in an embed

## Data Models
None required (stateless feature, reads from Discord session)

## Business Logic
- Show bot username and avatar
- Display number of servers (guilds)
- Show uptime since bot started
- Display Go version and bot version
- Public response (visible to all)

## Examples

### Example 1: User Checks Bot Info
```
User: /botinfo
Bot: [Embed showing:]
     Name: welcomebot Bot
     Version: 1.0.0
     Servers: 5
     Uptime: 2 days, 3 hours
     Language: Go 1.24
```

## Technical Requirements
- Track bot start time
- Read guild count from Discord session
- Show basic system information
- Public response (not ephemeral)

