CREATE TABLE comics (
  id          integer PRIMARY KEY,
  url         text    NOT NULL,
  title       text[]  NOT NULL  DEFAULT '{}',
  transcript  text[]  NOT NULL  DEFAULT '{}',
  alt         text[]  NOT NULL  DEFAULT '{}'
);

-- maybe overkill for small amount of data, but we have mainly READ load
CREATE INDEX idx_comics_title       ON comics USING GIN (title);
CREATE INDEX idx_comics_alt         ON comics USING GIN (alt);
CREATE INDEX idx_comics_transcript  ON comics USING GIN (transcript);
