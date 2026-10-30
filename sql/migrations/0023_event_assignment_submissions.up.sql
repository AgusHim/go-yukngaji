-- File tugas disimpan di Google Drive; aplikasi hanya menyimpan tautan publik.
CREATE TABLE IF NOT EXISTS event_assignment_submissions (
  id varchar PRIMARY KEY,
  episode_id varchar NOT NULL REFERENCES event_episodes(id) ON DELETE CASCADE,
  user_id varchar NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  file_type varchar NOT NULL CHECK (file_type IN ('document', 'video', 'image')),
  drive_url text NOT NULL,
  note text NOT NULL DEFAULT '',
  created_at timestamp NOT NULL DEFAULT now(),
  updated_at timestamp NOT NULL DEFAULT now(),
  UNIQUE (episode_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_assignment_submissions_user ON event_assignment_submissions(user_id);
