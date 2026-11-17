-- Match thumbnail backfill's ID cursor and exclude records that cannot be decoded.
DROP INDEX images_thumb_pending;
CREATE INDEX images_thumb_pending ON images(state, id)
    WHERE state = 'active' AND broken_at IS NULL AND thumb = '';
