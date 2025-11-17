# Caching Guidelines

## Overview

Caching is used for **performance** and **reducing database load**. All caching must follow strict guild-awareness rules.

---

## ⚠️ CRITICAL RULES

### Rule 1: Cache Keys MUST Include guild_id

**WRONG:**
```go
cacheKey := "config:" + channelID  ❌
```

**CORRECT:**
```go
cacheKey := fmt.Sprintf("welcomebot:config:%s:%s", guildID, channelID)  ✅
```

### Rule 2: Cache Key Pattern

```
welcomebot:{feature}:{guild_id}:{resource_id}
```

**Examples:**
```go
// Language
"welcomebot:i18n:guild:123456789"

// Gender roles
"welcomebot:gender:123456789"

// Room config
"welcomebot:rooms:123456789:987654321"
```

### Rule 3: Always Handle Cache Errors Gracefully

**Cache failures should NOT break functionality:**

```go
// WRONG: Fail if cache fails
config, err := cache.GetJSON(ctx, key, &data)
if err != nil {
    return err  ❌ // Breaks if cache is down!
}

// CORRECT: Fallback to database
var config Config
err := cache.GetJSON(ctx, key, &config)
if err != nil {
    // Cache miss or error - fetch from database
    config, err = f.db.GetConfig(ctx, guildID)
    if err != nil {
        return err
    }
    // Try to cache for next time (but don't fail if this errors)
    cache.SetJSON(ctx, key, config, ttl)
}
```

---

## TTL Guidelines

### When to Use What TTL

| Data Type | TTL | Constant | Reason |
|-----------|-----|----------|--------|
| **Indefinite** | `0` | N/A | Rarely changes, manually invalidated |
| **Long** | 2 hours | `shared.TTLLong` | Stable config |
| **Medium** | 30 min | `shared.TTLMedium` | Moderately stable |
| **Short** | 5 min | `shared.TTLShort` | Frequently changing |

### Examples

**Indefinite (TTL = 0):**
```go
// Language preference (only changes when admin updates)
cache.SetJSON(ctx, key, language, 0)

// Gender roles (only changes when admin updates)
cache.SetJSON(ctx, key, genderConfig, 0)

// Admin role config (only changes when admin updates)
cache.SetJSON(ctx, key, adminRole, 0)
```

**Long (2 hours):**
```go
// Room configurations
cache.SetJSON(ctx, key, roomConfig, shared.TTLLong)

// Feature settings
cache.SetJSON(ctx, key, settings, shared.TTLLong)
```

**Medium (30 minutes):**
```go
// User preferences
cache.SetJSON(ctx, key, userPrefs, shared.TTLMedium)

// Active sessions
cache.SetJSON(ctx, key, session, shared.TTLMedium)
```

**Short (5 minutes):**
```go
// Dynamic data
cache.SetJSON(ctx, key, stats, shared.TTLShort)

// Rate limiting
cache.Set(ctx, key, "1", shared.TTLShort)
```

---

## Pattern: Read-Through Cache

**Standard pattern for all features:**

```go
func (f *Feature) GetConfig(ctx context.Context, guildID string) (*Config, error) {
    cacheKey := fmt.Sprintf("welcomebot:feature:%s", guildID)
    
    // 1. Try cache first
    var config Config
    if err := f.cache.GetJSON(ctx, cacheKey, &config); err == nil {
        return &config, nil  // Cache hit!
    }
    
    // 2. Cache miss - fetch from database
    query := "SELECT * FROM configs WHERE guild_id = $1"
    row := f.db.QueryRow(ctx, query, guildID)
    
    if err := row.Scan(&config.Field1, &config.Field2); err != nil {
        return nil, fmt.Errorf("query config: %w", err)
    }
    
    // 3. Populate cache for next time (best effort)
    f.cache.SetJSON(ctx, cacheKey, &config, shared.TTLLong)
    
    return &config, nil
}
```

---

## Pattern: Write-Through Cache

**When saving data:**

```go
func (f *Feature) SaveConfig(ctx context.Context, guildID string, config *Config) error {
    // 1. Save to database first (source of truth)
    query := "INSERT INTO configs (...) VALUES (...) ON CONFLICT UPDATE ..."
    _, err := f.db.Exec(ctx, query, guildID, config.Value)
    if err != nil {
        return fmt.Errorf("save config: %w", err)
    }
    
    // 2. Update cache (best effort, don't fail if cache fails)
    cacheKey := fmt.Sprintf("welcomebot:feature:%s", guildID)
    if err := f.cache.SetJSON(ctx, cacheKey, config, 0); err != nil {
        f.logger.Warn("failed to update cache", "error", err)
        // Don't return error - database is saved, cache is optional
    }
    
    return nil
}
```

---

## Cache Invalidation

### Manual Invalidation

When config is deleted or changed significantly:

```go
func (f *Feature) DeleteConfig(ctx context.Context, guildID string) error {
    // 1. Delete from database
    query := "DELETE FROM configs WHERE guild_id = $1"
    _, err := f.db.Exec(ctx, query, guildID)
    if err != nil {
        return err
    }
    
    // 2. Invalidate cache
    cacheKey := fmt.Sprintf("welcomebot:feature:%s", guildID)
    f.cache.Delete(ctx, cacheKey)  // Best effort
    
    return nil
}
```

### TTL-Based Invalidation

Let Redis expire naturally for non-critical data:

```go
// Set with TTL - auto-expires
cache.SetJSON(ctx, key, data, 30*time.Minute)
// No manual invalidation needed
```

---

## What to Cache

### ✅ DO Cache:

- Configuration data (language, gender roles, admin roles)
- Frequently read, rarely updated data
- Data that's expensive to query
- Guild settings
- User preferences (if frequently accessed)

### ❌ DON'T Cache:

- Real-time data (voice state, online users)
- Data that changes frequently
- One-time use data
- Sensitive data with short validity
- Data only used once per request

---

## Error Handling

### Cache Failures Should NOT Break Features

```go
// CORRECT Pattern:
var data Data

// Try cache
err := cache.GetJSON(ctx, key, &data)
if err != nil {
    // Cache failed - fetch from database (don't return error)
    data, err = f.fetchFromDB(ctx, guildID)
    if err != nil {
        return err  // NOW return error (database is critical)
    }
}

// WRONG Pattern:
data, err := cache.GetJSON(ctx, key, &data)
if err != nil {
    return err  ❌ // Feature breaks if cache is down!
}
```

### Logging Cache Failures

```go
if err := cache.SetJSON(ctx, key, data, ttl); err != nil {
    f.logger.Warn("cache write failed",
        "key", key,
        "error", err,
    )
    // Continue - don't fail the operation
}
```

---

## Current Usage Examples

### Language (Indefinite)

```go
// Save
cache.Set(ctx, "welcomebot:i18n:guild:"+guildID, langCode, 0)

// Get with fallback
lang, err := cache.Get(ctx, key)
if err != nil {
    // Fallback to database
    lang, err = db.QueryLanguage(ctx, guildID)
}
```

### Gender Roles (Indefinite)

```go
// Save
cache.SetJSON(ctx, "welcomebot:gender:"+guildID, config, 0)

// Get with fallback
var config GenderConfig
if err := cache.GetJSON(ctx, key, &config); err != nil {
    // Fallback to database
    config, err = f.getFromDB(ctx, guildID)
}
```

---

## Checklist

When implementing caching:

- [ ] Cache key includes `guild_id`
- [ ] Cache key follows pattern: `welcomebot:{feature}:{guild_id}:{resource}`
- [ ] Appropriate TTL chosen (or 0 for indefinite)
- [ ] Cache errors don't break functionality
- [ ] Database is source of truth
- [ ] Cache is write-through (save DB first, then cache)
- [ ] Cache failures are logged but not fatal
- [ ] Cache invalidation on delete/update

---

## Summary

**Golden Rules:**

1. **Guild-Aware**: All keys include `guild_id`
2. **Database First**: DB is source of truth, cache is optimization
3. **Fail Gracefully**: Cache errors → fallback to DB
4. **Choose TTL Wisely**: Indefinite for config, TTL for dynamic data
5. **Best Effort**: Cache failures are warnings, not errors

**Pattern:**
```go
// Read: Cache → Database → Cache (populate)
// Write: Database → Cache (update)
// Delete: Database → Cache (invalidate)
```

