# Feature: Nested Menu System (3-Tier)

## User-Facing Description
Hierarchical navigation system for organizing bot features across multiple levels.

## Menu Structure

### Tier 1: Main Categories
- [👑 Admin] - Admin-only, leads to sub-menu
- [😊 User Features] - Public, direct features or sub-menu
- [📊 Information] - Public, direct features

### Tier 2: Sub-Categories (for Admin)
- Admin → Configuration (setup features)
- Admin → Tools (admin utilities)
- Admin → Stats (admin analytics)

### Tier 3: Features
- Admin → Configuration → [🚻 Set Gender Roles]
- Admin → Configuration → [🌐 Language Settings]
- Admin → Tools → [🗑️ Delete Messages]

## Navigation Flow

```
/menu → Main (Tier 1)
  ├─ [👑 Admin] → Admin Sub-Menu (Tier 2)
  │   ├─ [⚙️ Configuration] → Config Features (Tier 3)
  │   │   ├─ [🚻 Set Gender Roles] → Feature
  │   │   └─ [🌐 Language] → Feature
  │   └─ [🛠️ Tools] → Tool Features (Tier 3)
  │       └─ [🗑️ Delete Messages] → Feature
  │
  └─ [📊 Information] → Info Features (Tier 3)
      ├─ [🏓 Ping] → Feature
      └─ [ℹ️ Bot Info] → Feature
```

## CustomID Pattern

```
Main menu: "menu:main"
Category: "menu:category:admin"
Sub-category: "menu:subcategory:admin:configuration"
Feature: "menu:feature:admin:configuration:gender"
Back navigation: "menu:back:admin" or "menu:back:main"
```

## Business Logic

- Tier 1: Show main categories (admin-only filtered)
- Tier 2: Show sub-categories for selected main category
- Tier 3: Show features for selected sub-category
- Back button at each level (except main)
- Stateless navigation (state in CustomID)
- Permission filtering at all levels

## Data Models

None (uses feature registry)

## Technical Requirements

- Stateless (CustomID-based navigation)
- Permission-aware (hide admin sections from regular users)
- Breadcrumb tracking (for back navigation)
- Guild-aware (permissions checked per-guild)
- Ephemeral messages
- i18n for all labels

