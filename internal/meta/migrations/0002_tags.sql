CREATE TABLE image_tags (
    image_id TEXT NOT NULL REFERENCES images(id),
    tag      TEXT NOT NULL,
    PRIMARY KEY (image_id, tag)
) STRICT;
CREATE INDEX image_tags_tag ON image_tags(tag, image_id);
