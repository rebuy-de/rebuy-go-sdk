CREATE TABLE IF NOT EXISTS events (
    id    UInt64,
    name  String,
    ts    DateTime
) ENGINE = MergeTree
ORDER BY (ts, id);
