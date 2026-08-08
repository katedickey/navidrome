-- +goose Up
-- Grants read visibility on a playlist to users other than its owner, without
-- transferring ownership. Auto-imported playlists use it to become visible to the
-- users of the library they were imported from.
CREATE TABLE playlist_user (
    playlist_id VARCHAR(255) NOT NULL,
    user_id     VARCHAR(255) NOT NULL,
    PRIMARY KEY (playlist_id, user_id),
    FOREIGN KEY (playlist_id) REFERENCES playlist (id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES user (id) ON DELETE CASCADE
);
CREATE INDEX idx_playlist_user_playlist_id ON playlist_user (playlist_id);
CREATE INDEX idx_playlist_user_user_id ON playlist_user (user_id);

-- +goose Down
DROP INDEX IF EXISTS idx_playlist_user_user_id;
DROP INDEX IF EXISTS idx_playlist_user_playlist_id;
DROP TABLE IF EXISTS playlist_user;
