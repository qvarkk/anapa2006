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
    CHECK (status IN ('new', 'skipped', 'scheduled', 'sending', 'sent', 'fetch_error', 'archived')),
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

CREATE TABLE IF NOT EXISTS post_drafts (
  post_id INTEGER PRIMARY KEY,
  final_text TEXT NOT NULL,
  user_id INTEGER NOT NULL REFERENCES allowed_users(user_id),
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
  target_chat_id INTEGER NOT NULL,
  scheduled_at DATETIME NOT NULL,
  sent_at DATETIME,
  FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS pending_replies (
  chat_id INTEGER NOT NULL,
  prompt_message_id INTEGER NOT NULL,
  action TEXT NOT NULL
    CHECK (action IN ('awaiting_text', 'awaiting_schedule')),
  post_id INTEGER NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
  origin TEXT NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (chat_id, prompt_message_id)
);
