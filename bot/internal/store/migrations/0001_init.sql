CREATE TABLE IF NOT EXISTS allowed_users (
  user_id INTEGER PRIMARY KEY,
  username TEXT,
  lang TEXT NOT NULL DEFAULT 'ru' CHECK (lang in ('ru')),
  added_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS sources (
  id INTEGER PRIMARY KEY,
  channel_handle TEXT NOT NULL UNIQUE,
  active INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0, 1))
);

CREATE TABLE IF NOT EXISTS posts (
  id INTEGER PRIMARY KEY,
  source_id INTEGER NOT NULL REFERENCES sources(id),
  external_id TEXT NOT NULL,
  raw_text TEXT NOT NULL DEFAULT '',
  published_at DATETIME,
  fetched_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  status TEXT NOT NULL DEFAULT 'new' 
    CHECK (status IN ('new', 'skipped', 'scheduled', 'sending', 'sent', 'fetch_error', 'archived', 'error_sending')),
  retry_count INTEGER NOT NULL DEFAULT 0,
  UNIQUE(source_id, external_id)
);

CREATE TABLE IF NOT EXISTS post_media (
  id INTEGER PRIMARY KEY,
  post_id INTEGER NOT NULL REFERENCES posts(id),
  kind TEXT NOT NULL 
    CHECK (kind IN ('photo', 'video', 'document')),
  url TEXT NOT NULL,
  file_id TEXT,
  position INTEGER NOT NULL DEFAULT 0
);

-- Channels posts can be published to. Managed from the bot like sources; the
-- media cache channel stays an env variable for now.
CREATE TABLE IF NOT EXISTS target_channels (
  id INTEGER PRIMARY KEY,
  chat_id INTEGER NOT NULL UNIQUE,
  handle TEXT UNIQUE,
  title TEXT NOT NULL DEFAULT '',
  added_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS post_drafts (
  post_id INTEGER PRIMARY KEY,
  final_text TEXT NOT NULL,
  target_channel_id INTEGER REFERENCES target_channels(id) ON DELETE SET NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS draft_media (
  id INTEGER PRIMARY KEY,
  draft_id INTEGER NOT NULL,
  kind TEXT NOT NULL 
    CHECK (kind IN ('photo', 'video', 'document')),
  origin_media_id REFERENCES post_media(id),
  file_id TEXT,
  url TEXT,
  position INTEGER NOT NULL DEFAULT 0,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE (draft_id, position),
  FOREIGN KEY (draft_id) REFERENCES post_drafts(post_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS schedule (
  post_id INTEGER PRIMARY KEY,
  scheduled_at DATETIME NOT NULL,
  sent_at DATETIME,
  sent_message_id INTEGER,
  last_error TEXT,
  FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS pending_replies (
  chat_id INTEGER NOT NULL,
  prompt_message_id INTEGER NOT NULL,
  action TEXT NOT NULL
    CHECK (action IN ('awaiting_text', 'awaiting_schedule', 'awaiting_channel_handle', 'awaiting_target_handle', 'awaiting_user_ref', 'awaiting_media_add_position', 'awaiting_media_add', 'awaiting_media_substitute')),
  -- NULL for prompts that are not about a post
  post_id INTEGER REFERENCES posts(id) ON DELETE CASCADE,
  media_id INTEGER,
  position INTEGER,
  origin TEXT NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (chat_id, prompt_message_id)
);
