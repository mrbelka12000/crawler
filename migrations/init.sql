CREATE TABLE my_app_db.main
(
    parent String,
    child String,
    content BYTEA,
    created_at DateTime
)
    ENGINE = MergeTree()
PRIMARY KEY (parent, child);


