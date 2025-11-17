# Feature: Ping Command

## User-Facing Description
A simple ping command to verify the bot is responsive and measure latency.

## Commands/Interactions
- `/ping`: Responds with "Pong!" and latency information

## Data Models
None required (stateless feature)

## Business Logic
- Calculate and display bot latency
- Show WebSocket heartbeat latency
- Ephemeral response (only visible to command user)

## Examples

### Example 1: Basic Usage
```
User: /ping
Bot: 🏓 Pong!
     Latency: 45ms
     API Latency: 120ms
```

## Technical Requirements
- Stateless (no database or cache needed)
- Ephemeral responses
- Measure Discord API latency

